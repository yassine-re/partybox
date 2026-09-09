package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"partybox/backend/internal/database"
	"partybox/backend/internal/handlers"
	"partybox/backend/internal/models"
	"partybox/backend/internal/realtime"
	"partybox/backend/internal/repositories"
	"partybox/backend/internal/services"
	"partybox/backend/internal/vision"
)

type fakeVision struct {
	calls    atomic.Int32
	validate func(context.Context, vision.ValidationRequest) (vision.ValidationResult, error)
}

func (f *fakeVision) Validate(ctx context.Context, in vision.ValidationRequest) (vision.ValidationResult, error) {
	f.calls.Add(1)
	if f.validate != nil {
		return f.validate(ctx, in)
	}
	return vision.ValidationResult{Verdict: vision.Valid, Confidence: .94, Reason: "Objet visible."}, nil
}

func proofApp(t *testing.T, validator vision.Validator) (*pgxpool.Pool, *services.Service, http.Handler) {
	t.Helper()
	pool, dir := isolatedDB(t)
	migrateAndSeed(t, pool, dir)
	service := &services.Service{Repo: &repositories.Repository{Pool: pool}, Vision: validator, AI: &fakeMissionGenerator{}}
	gin.SetMode(gin.TestMode)
	rt := realtime.NewServer("http://localhost:3000", service.AuthorizeRealtime)
	t.Cleanup(rt.Close)
	return pool, service, handlers.Router(service, rt, "http://localhost:3000")
}

func proofGame(t *testing.T, router http.Handler, mode models.GameMode) (models.Session, models.Session) {
	t.Helper()
	host := request[models.Session](t, router, "POST", "/api/boxes/PB001/games", "", map[string]any{"name": "Proof test", "player_name": "Host", "mode": mode}, 201)
	guest := request[models.Session](t, router, "POST", "/api/games/"+host.Game.ID+"/join", "", map[string]string{"name": "Guest"}, 201)
	request[any](t, router, "POST", "/api/games/"+host.Game.ID+"/start", host.Token, nil, 200)
	return host, guest
}

func proofImage(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, image.NewRGBA(image.Rect(0, 0, 12, 12))); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func uploadProof(router http.Handler, token, id string, data []byte) *httptest.ResponseRecorder {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	w.WriteField("assignment_id", id)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="image"; filename="untrusted.jpg"`)
	header.Set("Content-Type", "image/jpeg") // Deliberately not the PNG's actual type.
	part, _ := w.CreatePart(header)
	part.Write(data)
	w.Close()
	req := httptest.NewRequest("POST", "/api/players/me/mission/proof", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res
}

func expectProof(t *testing.T, res *httptest.ResponseRecorder, status int) services.ProofResponse {
	t.Helper()
	if res.Code != status {
		t.Fatalf("got %d want %d: %s", res.Code, status, res.Body.String())
	}
	var out services.ProofResponse
	if status == 200 {
		if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func TestProofPreflightAndFallback(t *testing.T) {
	for _, mode := range []models.GameMode{models.ModeSecretMissions, models.ModeChaos, models.ModeTreasureHunt} {
		t.Run(string(mode), func(t *testing.T) {
			fake := &fakeVision{}
			_, service, router := proofApp(t, fake)
			host, guest := proofGame(t, router, mode)
			mission := missionFor(t, router, host)
			data := proofImage(t)
			status := request[services.ProofStatus](t, router, "GET", "/api/players/me/mission/proof", host.Token, nil, 200)
			if status.Available != (mode == models.ModeTreasureHunt) {
				t.Fatal("wrong availability")
			}
			expectProof(t, uploadProof(router, "", mission.ID, data), 401)
			expectProof(t, uploadProof(router, guest.Token, mission.ID, data), 404)
			expectProof(t, uploadProof(router, host.Token, "bad", data), 400)
			expectProof(t, uploadProof(router, host.Token, "00000000-0000-0000-0000-000000000000", data), 404)
			if mode != models.ModeTreasureHunt {
				expectProof(t, uploadProof(router, host.Token, mission.ID, data), 409)
				_, result := completeFor(t, router, host)
				if result.AwardedPoints != mission.Points {
					t.Fatal("other mode changed")
				}
			} else {
				expectProof(t, uploadProof(router, host.Token, mission.ID, []byte("not an image")), 400)
				expectProof(t, uploadProof(router, host.Token, mission.ID, data[:40]), 400)
				expectProof(t, uploadProof(router, host.Token, mission.ID, make([]byte, vision.MaxImageBytes+1)), 413)
				request[any](t, router, "POST", "/api/players/me/mission/complete", host.Token, map[string]string{"assignment_id": mission.ID}, 409)
				if fake.calls.Load() != 0 {
					t.Fatal("invalid input reached provider")
				}
				service.Vision = nil
				status = request[services.ProofStatus](t, router, "GET", "/api/players/me/mission/proof", host.Token, nil, 200)
				if status.Available {
					t.Fatal("fallback unavailable")
				}
				_, result := completeFor(t, router, host)
				if result.AwardedPoints != mission.Points {
					t.Fatal("manual fallback failed")
				}
			}
			if fake.calls.Load() != 0 {
				t.Fatal("preflight called provider")
			}
		})
	}
}

func TestProofInactiveAssignmentsAndMalformedProvider(t *testing.T) {
	fake := &fakeVision{validate: func(context.Context, vision.ValidationRequest) (vision.ValidationResult, error) {
		return vision.ValidationResult{Verdict: vision.Valid, Confidence: 2, Reason: "Bad provider confidence"}, nil
	}}
	pool, _, router := proofApp(t, fake)
	host, _ := proofGame(t, router, models.ModeTreasureHunt)
	mission := missionFor(t, router, host)
	data := proofImage(t)
	for _, state := range []string{"completed", "cancelled"} {
		mustExec(t, pool, `UPDATE player_missions SET status=$2,completed_at=CASE WHEN $2='completed' THEN now() ELSE NULL END WHERE id=$1`, mission.ID, state)
		expectProof(t, uploadProof(router, host.Token, mission.ID, data), 409)
	}
	mustExec(t, pool, `UPDATE player_missions SET status='assigned' WHERE id=$1`, mission.ID)
	for _, state := range []string{"lobby", "ended"} {
		mustExec(t, pool, `UPDATE games SET status=$2,ended_at=CASE WHEN $2='ended' THEN now() ELSE NULL END WHERE id=$1`, host.Game.ID, state)
		expectProof(t, uploadProof(router, host.Token, mission.ID, data), 409)
	}
	if fake.calls.Load() != 0 {
		t.Fatal("inactive assignment reached provider")
	}
	mustExec(t, pool, `UPDATE games SET status='playing' WHERE id=$1`, host.Game.ID)
	expectProof(t, uploadProof(router, host.Token, mission.ID, data), 503)
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM mission_proofs WHERE assignment_id=$1`, mission.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("invalid provider output persisted")
	}
	p := request[models.Player](t, router, "GET", "/api/players/me", host.Token, nil, 200)
	if p.Score != 0 {
		t.Fatal("malformed provider output scored")
	}
}

func TestProofVerdictsPersistenceAndRealtime(t *testing.T) {
	fake := &fakeVision{}
	pool, _, router := proofApp(t, fake)
	host, guest := proofGame(t, router, models.ModeTreasureHunt)
	mission := missionFor(t, router, host)
	server := httptest.NewServer(router)
	defer server.Close()
	ticket := request[struct {
		Ticket string `json:"ticket"`
	}](t, router, "POST", "/api/games/"+host.Game.ID+"/ws-ticket", guest.Token, nil, 201)
	headers := http.Header{"Origin": []string{"http://localhost:3000"}}
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/api/games/"+host.Game.ID+"/ws?ticket="+ticket.Ticket, headers)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	data := proofImage(t)
	for _, verdict := range []vision.Verdict{vision.Invalid, vision.Uncertain, vision.Valid} {
		fake.validate = func(ctx context.Context, in vision.ValidationRequest) (vision.ValidationResult, error) {
			if in.MissionText != mission.Text || in.MediaType != "image/png" || !bytes.Equal(data, in.Image) {
				t.Error("input tampered")
			}
			return vision.ValidationResult{Verdict: verdict, Confidence: .9, Reason: "Raison privée."}, nil
		}
		result := expectProof(t, uploadProof(router, host.Token, mission.ID, data), 200)
		if result.Proof.Verdict != string(verdict) {
			t.Fatal("verdict changed")
		}
		if verdict != vision.Valid {
			if result.Completion != nil {
				t.Fatal("invalid/uncertain completed")
			}
			player := request[models.Player](t, router, "GET", "/api/players/me", host.Token, nil, 200)
			if player.Score != 0 || missionFor(t, router, host).ID != mission.ID {
				t.Fatal("rejected proof changed game")
			}
		} else if result.Completion == nil || result.Completion.AwardedPoints != mission.Points || result.Completion.Player.Score != mission.Points || result.Completion.Mission == nil || result.Completion.Mission.ID == mission.ID {
			t.Fatalf("valid proof did not use completion: %+v", result)
		}
	}
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, event, err := ws.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(event, []byte(`"type":"mission_completed"`)) || bytes.Contains(event, []byte("verdict")) || bytes.Contains(event, []byte("Raison")) {
		t.Fatalf("bad realtime signal: %s", event)
	}
	for range 2 {
		retry := expectProof(t, uploadProof(router, host.Token, mission.ID, data), 200)
		if retry.Completion == nil || !retry.Completion.AlreadyCompleted || retry.Completion.AwardedPoints != 0 {
			t.Fatal("retry scored twice")
		}
	}
	if fake.calls.Load() != 3 {
		t.Fatal("retry paid twice")
	}
	var proofs, events, completions, columns int
	ctx := context.Background()
	pool.QueryRow(ctx, `SELECT count(*) FROM mission_proofs WHERE assignment_id=$1`, mission.ID).Scan(&proofs)
	pool.QueryRow(ctx, `SELECT count(*) FROM game_events WHERE game_id=$1 AND type='mission_proof_evaluated'`, host.Game.ID).Scan(&events)
	pool.QueryRow(ctx, `SELECT count(*) FROM game_events WHERE game_id=$1 AND type='mission_completed'`, host.Game.ID).Scan(&completions)
	pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='mission_proofs'`).Scan(&columns)
	if proofs != 3 || events != 3 || completions != 1 || columns != 6 {
		t.Fatalf("proofs=%d events=%d completions=%d columns=%d", proofs, events, completions, columns)
	}
	var payload string
	if err := pool.QueryRow(ctx, `SELECT payload::text FROM game_events WHERE game_id=$1 AND type='mission_proof_evaluated' LIMIT 1`, host.Game.ID).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(payload, "reason") || strings.Contains(payload, "image") {
		t.Fatal("private data in events")
	}
}

func TestProofAttemptsAreDurableAndBounded(t *testing.T) {
	for _, providerError := range []bool{false, true} {
		t.Run(map[bool]string{false: "concurrent invalid", true: "provider failures"}[providerError], func(t *testing.T) {
			fake := &fakeVision{validate: func(context.Context, vision.ValidationRequest) (vision.ValidationResult, error) {
				if providerError {
					return vision.ValidationResult{}, errors.New("private-provider-data")
				}
				return vision.ValidationResult{Verdict: vision.Invalid, Confidence: .8, Reason: "Pas encore."}, nil
			}}
			pool, _, router := proofApp(t, fake)
			host, _ := proofGame(t, router, models.ModeTreasureHunt)
			mission := missionFor(t, router, host)
			data := proofImage(t)
			var wg sync.WaitGroup
			responses := make(chan *httptest.ResponseRecorder, 10)
			for range 10 {
				wg.Go(func() { responses <- uploadProof(router, host.Token, mission.ID, data) })
			}
			wg.Wait()
			close(responses)
			limited, attempted := 0, 0
			for response := range responses {
				if strings.Contains(response.Body.String(), "private-provider-data") {
					t.Fatal("provider data leaked")
				}
				if response.Code == 429 {
					limited++
				} else if response.Code == 200 && !providerError || response.Code == 503 && providerError {
					attempted++
				} else {
					t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
				}
			}
			if limited != 5 || attempted != 5 || fake.calls.Load() != 5 {
				t.Fatalf("limit race: limited=%d attempted=%d calls=%d", limited, attempted, fake.calls.Load())
			}
			restarted := &services.Service{Repo: &repositories.Repository{Pool: pool}, Vision: fake}
			_, err := restarted.SubmitMissionProof(context.Background(), host.Player, mission.ID, data)
			if !errors.Is(err, services.ErrProofLimit) {
				t.Fatalf("restart reset budget: %v", err)
			}
			status := request[services.ProofStatus](t, router, "GET", "/api/players/me/mission/proof", host.Token, nil, 200)
			if status.Attempts != 5 || status.RemainingAttempts != 0 {
				t.Fatalf("bad quota: %+v", status)
			}
			p := request[models.Player](t, router, "GET", "/api/players/me", host.Token, nil, 200)
			if p.Score != 0 {
				t.Fatal("failed calls scored")
			}
		})
	}
}

func TestProofProviderDoesNotHoldTransactionAndEndWins(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	fake := &fakeVision{validate: func(context.Context, vision.ValidationRequest) (vision.ValidationResult, error) {
		close(entered)
		<-release
		return vision.ValidationResult{Verdict: vision.Valid, Confidence: 1, Reason: "Visible."}, nil
	}}
	pool, service, router := proofApp(t, fake)
	host, _ := proofGame(t, router, models.ModeTreasureHunt)
	mission := missionFor(t, router, host)
	data := proofImage(t)
	result := make(chan *httptest.ResponseRecorder, 1)
	go func() { result <- uploadProof(router, host.Token, mission.ID, data) }()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("provider not entered")
	}
	if pool.Stat().AcquiredConns() != 0 {
		t.Error("connection held during provider call")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	err := service.End(ctx, host.Game.ID, host.Player)
	cancel()
	close(release)
	if err != nil {
		t.Fatalf("game lock held during vision: %v", err)
	}
	expectProof(t, <-result, 409)
	p := request[models.Player](t, router, "GET", "/api/players/me", host.Token, nil, 200)
	if p.Score != 0 {
		t.Fatal("awarded after end")
	}
	expectProof(t, uploadProof(router, host.Token, mission.ID, data), 409)
	if fake.calls.Load() != 1 {
		t.Fatal("ended game called provider")
	}
}

func TestPhotoMigrationExistingDatabaseAndRollback(t *testing.T) {
	pool, dir := isolatedDB(t)
	legacy := t.TempDir()
	for _, name := range []string{"001_initial", "002_game_modes", "003_game_events", "004_ai_missions", "005_chaos_mode"} {
		file := name + ".up.sql"
		data, err := os.ReadFile(filepath.Join(dir, "migrations", file))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(legacy, file), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	if err := database.Migrate(ctx, pool, legacy); err != nil {
		t.Fatal(err)
	}
	if err := database.Seed(ctx, pool, filepath.Join(dir, "seed.sql")); err != nil {
		t.Fatal(err)
	}
	service := &services.Service{Repo: &repositories.Repository{Pool: pool}}
	host, err := service.Create(ctx, "PB001", "Existing", "Host", models.ModeTreasureHunt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Join(ctx, host.Game.ID, "Guest"); err != nil {
		t.Fatal(err)
	}
	if err = service.Start(ctx, host.Game.ID, host.Player); err != nil {
		t.Fatal(err)
	}
	mission, err := service.Repo.Mission(ctx, host.Player)
	if err != nil {
		t.Fatal(err)
	}
	migrateAndSeed(t, pool, dir)
	a, err := service.Repo.ProofAssignment(ctx, host.Player, mission.ID)
	if err != nil || a.Attempts != 0 || a.Status != "assigned" {
		t.Fatalf("existing assignment lost: %+v %v", a, err)
	}
	down, err := os.ReadFile(filepath.Join(dir, "migrations", "006_treasure_photo_validation.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, pool, string(down))
	completion, err := service.Complete(ctx, host.Player, mission.ID)
	if err != nil || completion.AwardedPoints != mission.Points {
		t.Fatalf("rollback changed gameplay: %+v %v", completion, err)
	}
}
