package handlers_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"partybox/backend/internal/database"
	"partybox/backend/internal/handlers"
	"partybox/backend/internal/models"
	"partybox/backend/internal/realtime"
	"partybox/backend/internal/repositories"
	"partybox/backend/internal/services"
)

// Every test owns an isolated PostgreSQL schema; no development game is changed.
func isolatedDB(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx := context.Background()
	admin, err := database.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("partybox_test_%d", time.Now().UnixNano())
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Errorf("cleanup: %v", err)
		}
		admin.Close()
	})
	return pool, filepath.Join("..", "..", "..", "database")
}

func mustExec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func migrateAndSeed(t *testing.T, pool *pgxpool.Pool, dir string) {
	t.Helper()
	ctx := context.Background()
	// Run twice to exercise migration tracking and seed idempotency.
	for range 2 {
		if err := database.Migrate(ctx, pool, filepath.Join(dir, "migrations")); err != nil {
			t.Fatal(err)
		}
		if err := database.Seed(ctx, pool, filepath.Join(dir, "seed.sql")); err != nil {
			t.Fatal(err)
		}
	}
	var secret, treasure int
	if err := pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE mode='secret_missions'), count(*) FILTER (WHERE mode='treasure_hunt') FROM missions`).Scan(&secret, &treasure); err != nil {
		t.Fatal(err)
	}
	if secret != 18 || treasure != 15 {
		t.Fatalf("catalogs: secret=%d treasure=%d", secret, treasure)
	}
}

func call(router http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response
}

func request[T any](t *testing.T, router http.Handler, method, path, token string, body any, status int) T {
	t.Helper()
	response := call(router, method, path, token, body)
	if response.Code != status {
		t.Fatalf("%s %s: got %d, want %d: %s", method, path, response.Code, status, response.Body.String())
	}
	var result T
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestGameModesOnExistingDatabase(t *testing.T) {
	pool, dir := isolatedDB(t)
	ctx := context.Background()
	legacyDir := t.TempDir()
	initial, err := os.ReadFile(filepath.Join(dir, "migrations", "001_initial.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(legacyDir, "001_initial.up.sql"), initial, 0600); err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(ctx, pool, legacyDir); err != nil {
		t.Fatal(err)
	}
	legacyToken := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))
	mustExec(t, pool, `INSERT INTO boxes(id,name) VALUES('LEGACY','Box historique');
		INSERT INTO missions(id,text,points,category,difficulty) VALUES(1,'Mission historique conservée',10,'Conversation',1);
		INSERT INTO games(id,box_id,name,status,started_at) VALUES('00000000-0000-0000-0000-000000000001','LEGACY','Partie historique','playing',now());`)
	mustExec(t, pool, `INSERT INTO players(id,game_id,name,score,is_host,token_hash) VALUES('00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001','Hôte historique',10,true,$1)`, services.TokenHash(legacyToken))
	mustExec(t, pool, `INSERT INTO player_missions(id,player_id,mission_id,status,completed_at) VALUES('00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000002',1,'completed',now());
		INSERT INTO player_missions(id,player_id,mission_id) VALUES('00000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000002',1);`)
	gameModesMigration, err := os.ReadFile(filepath.Join(dir, "migrations", "002_game_modes.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(legacyDir, "002_game_modes.up.sql"), gameModesMigration, 0600); err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(ctx, pool, legacyDir); err != nil {
		t.Fatal(err)
	}
	var appliedBeforeEvents int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&appliedBeforeEvents); err != nil {
		t.Fatal(err)
	}
	if appliedBeforeEvents != 2 {
		t.Fatalf("got %d migrations before 003, want 2", appliedBeforeEvents)
	}
	migrateAndSeed(t, pool, dir)
	gin.SetMode(gin.TestMode)
	service := &services.Service{Repo: &repositories.Repository{Pool: pool}}
	realtimeServer := realtime.NewServer("http://localhost:3000", service.AuthorizeRealtime)
	t.Cleanup(realtimeServer.Close)
	router := handlers.Router(service, realtimeServer, "http://localhost:3000")
	t.Run("legacy_session_and_assignment_survive_001_002_to_003", func(t *testing.T) {
		p := request[models.Player](t, router, "GET", "/api/players/me", legacyToken, nil, 200)
		if p.Name != "Hôte historique" || p.Score != 10 || p.CompletedMissions != 1 {
			t.Fatalf("legacy player changed: %+v", p)
		}
		g := request[models.Game](t, router, "GET", "/api/games/"+p.GameID, legacyToken, nil, 200)
		if g.Mode != models.ModeSecretMissions || g.Status != "playing" {
			t.Fatalf("legacy game changed: %+v", g)
		}
		m := request[struct{ Mission models.Mission }](t, router, "GET", "/api/players/me/mission", legacyToken, nil, 200).Mission
		if m.ID != "00000000-0000-0000-0000-000000000004" || m.Text != "Mission historique conservée" {
			t.Fatalf("legacy assignment changed: %+v", m)
		}
		result := request[models.Completion](t, router, "POST", "/api/players/me/mission/complete", legacyToken, map[string]string{"assignment_id": m.ID}, 200)
		if result.Player.Score != 20 || result.Mission == nil || result.Mission.MissionID == m.MissionID {
			t.Fatal("legacy completion failed")
		}
	})
	t.Run("invalid_modes_and_legacy_default", func(t *testing.T) {
		for _, mode := range []any{"unknown", "", "TREASURE_HUNT", " treasure_hunt", 12, []string{"treasure_hunt"}} {
			request[any](t, router, "POST", "/api/boxes/PB001/games", "", map[string]any{"name": "Invalid", "player_name": "Host", "mode": mode}, 400)
		}
		session := request[models.Session](t, router, "POST", "/api/boxes/PB001/games", "", map[string]string{"name": "Old client", "player_name": "Host"}, 201)
		if session.Game.Mode != models.ModeSecretMissions {
			t.Fatal("missing mode must preserve Secret Missions")
		}
		request[any](t, router, "POST", "/api/games/"+session.Game.ID+"/end", session.Token, nil, 200)
	})
	for _, mode := range []models.GameMode{models.ModeSecretMissions, models.ModeTreasureHunt} {
		t.Run(string(mode), func(t *testing.T) { exerciseMode(t, pool, router, mode) })
	}
	var crossed int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM player_missions pm JOIN players p ON p.id=pm.player_id JOIN games g ON g.id=p.game_id JOIN missions m ON m.id=pm.mission_id WHERE m.mode<>g.mode`).Scan(&crossed); err != nil {
		t.Fatal(err)
	}
	if crossed != 0 {
		t.Fatalf("%d assignments crossed modes", crossed)
	}
}

func exerciseMode(t *testing.T, pool *pgxpool.Pool, router http.Handler, mode models.GameMode) {
	t.Helper()
	host := request[models.Session](t, router, "POST", "/api/boxes/PB001/games", "", map[string]any{"name": string(mode), "player_name": "Host", "mode": mode}, 201)
	if host.Game.Mode != mode || !host.Player.IsHost {
		t.Fatal("mode or host missing")
	}
	base := "/api/games/" + host.Game.ID
	request[any](t, router, "POST", base+"/start", host.Token, nil, 409)
	guest := request[models.Session](t, router, "POST", base+"/join", "", map[string]string{"name": "Guest"}, 201)
	if guest.Game.Mode != mode || guest.Player.IsHost {
		t.Fatal("join changed mode or host")
	}
	request[any](t, router, "POST", base+"/start", guest.Token, nil, 403)
	request[any](t, router, "POST", base+"/start", host.Token, nil, 200)
	request[any](t, router, "POST", base+"/start", host.Token, nil, 200)
	late := request[models.Session](t, router, "POST", base+"/join", "", map[string]string{"name": "Late"}, 201)
	for _, session := range []models.Session{host, guest, late} {
		m := request[struct{ Mission *models.Mission }](t, router, "GET", "/api/players/me/mission", session.Token, nil, 200).Mission
		if m == nil {
			t.Fatal("missing assignment")
		}
		assertMissionMode(t, pool, m.MissionID, mode)
	}
	mission := request[struct{ Mission models.Mission }](t, router, "GET", "/api/players/me/mission", host.Token, nil, 200).Mission
	request[any](t, router, "POST", "/api/players/me/mission/complete", guest.Token, map[string]string{"assignment_id": mission.ID}, 404)
	// Duplicate submissions must not add points twice or assign two challenges.
	var wg sync.WaitGroup
	responses := make(chan *httptest.ResponseRecorder, 5)
	for range 5 {
		wg.Go(func() {
			responses <- call(router, "POST", "/api/players/me/mission/complete", host.Token, map[string]string{"assignment_id": mission.ID})
		})
	}
	wg.Wait()
	close(responses)
	awarded, duplicates := 0, 0
	for response := range responses {
		if response.Code != 200 {
			t.Fatalf("concurrent completion: %s", response.Body.String())
		}
		var result models.Completion
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		awarded += result.AwardedPoints
		if result.AlreadyCompleted {
			duplicates++
		}
	}
	if awarded != mission.Points || duplicates != 4 {
		t.Fatalf("duplicate points: awarded=%d duplicates=%d", awarded, duplicates)
	}
	// Walk past catalogue exhaustion: unseen first, no immediate repeats, mode isolation.
	catalogSize := 18
	if mode == models.ModeTreasureHunt {
		catalogSize = 15
	}
	seen := map[int]bool{mission.MissionID: true}
	score := mission.Points
	for i := 1; i <= catalogSize+2; i++ {
		next := request[struct{ Mission models.Mission }](t, router, "GET", "/api/players/me/mission", host.Token, nil, 200).Mission
		assertMissionMode(t, pool, next.MissionID, mode)
		if next.MissionID == mission.MissionID {
			t.Fatal("immediate repeat")
		}
		if i < catalogSize && seen[next.MissionID] {
			t.Fatal("repeated before all challenges were seen")
		}
		seen[next.MissionID] = true
		result := request[models.Completion](t, router, "POST", "/api/players/me/mission/complete", host.Token, map[string]string{"assignment_id": next.ID}, 200)
		score += next.Points
		if result.Player.Score != score || result.Mission == nil {
			t.Fatal("completion must atomically add points and assign next challenge")
		}
		mission = next
	}
	p := request[models.Player](t, router, "GET", "/api/players/me", host.Token, nil, 200)
	if p.ID != host.Player.ID || p.Score != score || p.CompletedMissions != catalogSize+3 {
		t.Fatal("session restoration lost progress")
	}
	ranking := request[struct{ Players []models.Player }](t, router, "GET", base+"/leaderboard", host.Token, nil, 200)
	if len(ranking.Players) != 3 || ranking.Players[0].ID != host.Player.ID || ranking.Players[0].Score != score {
		t.Fatal("ranking incorrect")
	}
	request[any](t, router, "POST", base+"/end", guest.Token, nil, 403)
	request[any](t, router, "POST", base+"/end", host.Token, nil, 200)
	request[any](t, router, "POST", base+"/join", "", map[string]string{"name": "Too late"}, 409)
	current := request[struct{ Mission models.Mission }](t, router, "GET", "/api/players/me/mission", host.Token, nil, 200).Mission
	request[any](t, router, "POST", "/api/players/me/mission/complete", host.Token, map[string]string{"assignment_id": current.ID}, 409)
	final := request[models.Game](t, router, "GET", base, host.Token, nil, 200)
	if final.Status != "ended" || final.Players[0].Score != score {
		t.Fatal("final scores changed")
	}
}

func assertMissionMode(t *testing.T, pool *pgxpool.Pool, id int, want models.GameMode) {
	t.Helper()
	var got models.GameMode
	if err := pool.QueryRow(context.Background(), `SELECT mode FROM missions WHERE id=$1`, id).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("mission %d belongs to %s, not %s", id, got, want)
	}
}

func TestGameModesFreshDatabaseAndRollback(t *testing.T) {
	pool, dir := isolatedDB(t)
	migrateAndSeed(t, pool, dir)
	down, err := os.ReadFile(filepath.Join(dir, "migrations", "002_game_modes.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, pool, string(down))
	var count int
	if err = pool.QueryRow(context.Background(), `SELECT count(*) FROM missions`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 18 {
		t.Fatalf("rollback lost Secret Missions: %d", count)
	}
	mustExec(t, pool, `DELETE FROM schema_migrations WHERE version='002_game_modes'`)
	migrateAndSeed(t, pool, dir)
	mustExec(t, pool, `INSERT INTO games(id,box_id,name,mode) VALUES(gen_random_uuid(),'PB001','Keep this game','treasure_hunt')`)
	if _, err = pool.Exec(context.Background(), string(down)); err == nil {
		t.Fatal("rollback must refuse to delete Treasure Hunt history")
	}
}

func TestAIMigrationRollbackAfterMissionAssignment(t *testing.T) {
	pool, dir := isolatedDB(t)
	migrateAndSeed(t, pool, dir)
	ctx := context.Background()

	var gameID, playerID string
	if err := pool.QueryRow(ctx, `INSERT INTO games(id,box_id,name,mode,status)
		VALUES(gen_random_uuid(),'PB001','Rollback IA','secret_missions','playing') RETURNING id`).Scan(&gameID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO players(id,game_id,name,token_hash,is_host)
		VALUES(gen_random_uuid(),$1,'Host rollback','rollback-token',true) RETURNING id`, gameID).Scan(&playerID); err != nil {
		t.Fatal(err)
	}
	var missionID int
	if err := pool.QueryRow(ctx, `INSERT INTO missions(text,points,category,difficulty,mode,source,game_id)
		VALUES('Mission IA à supprimer',10,'test',1,'secret_missions','ai',$1) RETURNING id`, gameID).Scan(&missionID); err != nil {
		t.Fatal(err)
	}
	mustExec(t, pool, `INSERT INTO player_missions(id,player_id,mission_id,status)
		VALUES(gen_random_uuid(),$1,$2,'assigned')`, playerID, missionID)

	down, err := os.ReadFile(filepath.Join(dir, "migrations", "004_ai_missions.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("004 rollback failed with an assigned AI mission: %v", err)
	}
	var columns int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns
		WHERE table_schema=current_schema() AND table_name='missions' AND column_name IN ('source','game_id')`).Scan(&columns); err != nil {
		t.Fatal(err)
	}
	if columns != 0 {
		t.Fatalf("004 rollback left %d AI columns", columns)
	}
}
