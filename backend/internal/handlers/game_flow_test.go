package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"partybox/backend/internal/handlers"
	"partybox/backend/internal/models"
	"partybox/backend/internal/realtime"
	"partybox/backend/internal/repositories"
	"partybox/backend/internal/services"
)

func flowRouter(t *testing.T) (*services.Service, http.Handler) {
	t.Helper()
	pool, dir := isolatedDB(t)
	migrateAndSeed(t, pool, dir)
	gin.SetMode(gin.TestMode)
	s := &services.Service{Repo: &repositories.Repository{Pool: pool}}
	rt := realtime.NewServer("http://localhost:3000", s.AuthorizeRealtime)
	t.Cleanup(rt.Close)
	return s, handlers.Router(s, rt, "http://localhost:3000")
}

func flowGame(t *testing.T, router http.Handler, box string, duration int, visibility, validation string) (models.Session, models.Session, string) {
	t.Helper()
	host := request[models.Session](t, router, "POST", "/api/boxes/"+box+"/games", "", map[string]any{
		"name": "Soirée", "player_name": "Host", "mode": "secret_missions", "duration_minutes": duration,
		"leaderboard_visibility": visibility, "validation_mode": validation,
	}, 201)
	base := "/api/games/" + host.Game.ID
	guest := request[models.Session](t, router, "POST", base+"/join", "", map[string]string{"name": "Guest"}, 201)
	request[any](t, router, "POST", base+"/start", host.Token, nil, 200)
	return host, guest, base
}

func flowMission(t *testing.T, router http.Handler, token string) models.Mission {
	t.Helper()
	return request[struct{ Mission models.Mission }](t, router, "GET", "/api/players/me/mission", token, nil, 200).Mission
}

func TestGameDurationsAndAutomaticFinish(t *testing.T) {
	for _, duration := range []int{5, 15, 30, 60, 0} {
		t.Run(fmt.Sprint(duration), func(t *testing.T) {
			s, router := flowRouter(t)
			now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
			s.Now = func() time.Time { return now }
			host, _, base := flowGame(t, router, "PB001", duration, "visible", "trust")
			g := request[models.Game](t, router, "GET", base, host.Token, nil, 200)
			if g.DurationMinutes != duration {
				t.Fatalf("duration=%d", g.DurationMinutes)
			}
			if duration == 0 {
				if g.EndsAt != nil {
					t.Fatal("infinite game has ends_at")
				}
				now = now.Add(14*time.Minute + 30*time.Second)
				mission := flowMission(t, router, host.Token)
				completion := request[models.Completion](t, router, "POST", "/api/players/me/mission/complete", host.Token, map[string]any{"assignment_id": mission.ID}, 200)
				if completion.AwardedPoints != mission.Points {
					t.Fatalf("infinite points=%d", completion.AwardedPoints)
				}
				request[any](t, router, "POST", base+"/end", host.Token, nil, 200)
			} else {
				want := now.Add(time.Duration(duration) * time.Minute)
				if g.EndsAt == nil || !g.EndsAt.Equal(want) {
					t.Fatalf("ends_at=%v want %v", g.EndsAt, want)
				}
				now = want
				changes, err := s.TickGames(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				if len(changes) != 1 || changes[0].Type != models.GameEventGameEnded {
					t.Fatalf("changes=%+v", changes)
				}
				mission := flowMission(t, router, host.Token)
				if call(router, "POST", "/api/players/me/mission/complete", host.Token, map[string]any{"assignment_id": mission.ID}).Code != 409 {
					t.Fatal("completed after deadline")
				}
			}
			g = request[models.Game](t, router, "GET", base, host.Token, nil, 200)
			if g.Status != "ended" || g.FinishedAt == nil {
				t.Fatalf("not finished: %+v", g)
			}
		})
	}
}

func TestFinalMinuteScoringAndHiddenResults(t *testing.T) {
	s, router := flowRouter(t)
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return now }
	host, guest, base := flowGame(t, router, "PB001", 15, "hidden", "trust")
	mission := flowMission(t, router, host.Token)
	if call(router, "GET", base+"/recap", guest.Token, nil).Code != 409 {
		t.Fatal("recap leaked before finish")
	}
	now = now.Add(13*time.Minute + 59*time.Second)
	first := request[models.Completion](t, router, "POST", "/api/players/me/mission/complete", host.Token, map[string]any{"assignment_id": mission.ID, "multiplier": 99, "points": 999999}, 200)
	if first.AwardedPoints != mission.Points {
		t.Fatalf("normal points=%d", first.AwardedPoints)
	}
	hidden := call(router, "GET", base+"/leaderboard", guest.Token, nil)
	if hidden.Code != 200 || strings.Contains(hidden.Body.String(), `"score":`+fmt.Sprint(first.Player.Score)) {
		t.Fatalf("hidden score leaked: %s", hidden.Body.String())
	}
	mission = flowMission(t, router, host.Token)
	now = now.Add(2 * time.Second)
	second := request[models.Completion](t, router, "POST", "/api/players/me/mission/complete", host.Token, map[string]any{"assignment_id": mission.ID, "multiplier": 99}, 200)
	if second.AwardedPoints != mission.Points*2 {
		t.Fatalf("final points=%d want %d", second.AwardedPoints, mission.Points*2)
	}
	now = now.Add(59 * time.Second)
	finished := request[models.Game](t, router, "GET", base, guest.Token, nil, 200)
	if finished.Status != "ended" || finished.Players[0].Score != first.AwardedPoints+second.AwardedPoints {
		t.Fatalf("final scoreboard=%+v", finished)
	}
	recap := request[models.FinalRecap](t, router, "GET", base+"/recap", guest.Token, nil, 200)
	if len(recap.Players) != 2 || len(recap.Players[0].Missions) < 2 {
		t.Fatalf("recap=%+v", recap)
	}
}

func TestManualFinishBeforeDeadline(t *testing.T) {
	s, router := flowRouter(t)
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return now }
	host, _, base := flowGame(t, router, "PB001", 30, "visible", "trust")
	now = now.Add(2 * time.Minute)
	request[any](t, router, "POST", base+"/end", host.Token, nil, 200)
	game := request[models.Game](t, router, "GET", base, host.Token, nil, 200)
	if game.Status != "ended" || game.FinishedAt == nil || !game.FinishedAt.Equal(now) {
		t.Fatalf("manual finish=%+v", game)
	}
	mission := flowMission(t, router, host.Token)
	if call(router, "POST", "/api/players/me/mission/complete", host.Token, map[string]any{"assignment_id": mission.ID}).Code != 409 {
		t.Fatal("completion accepted after manual finish")
	}
}

func TestPeerValidationIsolationAndExpiry(t *testing.T) {
	s, router := flowRouter(t)
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return now }
	host, guest, base := flowGame(t, router, "PB001", 15, "hidden", "peer")
	mission := flowMission(t, router, host.Token)
	pending := request[models.Completion](t, router, "POST", "/api/players/me/mission/complete", host.Token, map[string]any{"assignment_id": mission.ID}, 200)
	if !pending.ValidationPending || pending.Player.Score != 0 {
		t.Fatalf("not pending: %+v", pending)
	}
	list := request[struct{ Requests []models.ValidationRequest }](t, router, "GET", base+"/validations", guest.Token, nil, 200)
	if len(list.Requests) != 1 || list.Requests[0].MissionText != mission.Text {
		t.Fatalf("requests=%+v", list.Requests)
	}
	id := list.Requests[0].ID
	path := base + "/validations/" + id + "/resolve"
	if call(router, "POST", path, host.Token, map[string]bool{"approved": true}).Code != 403 {
		t.Fatal("self validation accepted")
	}
	if call(router, "POST", path, guest.Token, map[string]bool{"approved": false}).Code != 200 {
		t.Fatal("rejection failed")
	}
	if request[models.Player](t, router, "GET", "/api/players/me", host.Token, nil, 200).Score != 0 {
		t.Fatal("rejection scored")
	}
	if flowMission(t, router, host.Token).ID != mission.ID {
		t.Fatal("rejection replaced mission")
	}
	request[models.Completion](t, router, "POST", "/api/players/me/mission/complete", host.Token, map[string]any{"assignment_id": mission.ID}, 200)
	approvalResponse := call(router, "POST", path, guest.Token, map[string]bool{"approved": true})
	if approvalResponse.Code != 200 {
		t.Fatalf("approval: %s", approvalResponse.Body.String())
	}
	var approved models.Completion
	if err := json.Unmarshal(approvalResponse.Body.Bytes(), &approved); err != nil {
		t.Fatal(err)
	}
	if approved.AwardedPoints != mission.Points {
		t.Fatalf("awarded=%d", approved.AwardedPoints)
	}
	nextMission := flowMission(t, router, host.Token)
	if strings.Contains(approvalResponse.Body.String(), nextMission.Text) || strings.Contains(approvalResponse.Body.String(), `"score"`) {
		t.Fatalf("private data leaked to validator: %s", approvalResponse.Body.String())
	}
	if call(router, "POST", path, guest.Token, map[string]bool{"approved": true}).Code != 409 {
		t.Fatal("double validation succeeded")
	}
	if nextMission.ID == mission.ID {
		t.Fatal("no next mission")
	}
	// A second game cannot resolve a request from another game.
	mustExec(t, s.Repo.Pool, `INSERT INTO boxes(id,name) VALUES('PB902','Other')`)
	other, _, otherBase := flowGame(t, router, "PB902", 0, "visible", "peer")
	if call(router, "POST", otherBase+"/validations/"+id+"/resolve", other.Token, map[string]bool{"approved": true}).Code != 404 {
		t.Fatal("cross-game validation leaked")
	}
	// Pending validation expires without points when the host ends early.
	next := flowMission(t, router, host.Token)
	request[models.Completion](t, router, "POST", "/api/players/me/mission/complete", host.Token, map[string]any{"assignment_id": next.ID}, 200)
	pendingAfter := request[struct{ Requests []models.ValidationRequest }](t, router, "GET", base+"/validations", guest.Token, nil, 200)
	if len(pendingAfter.Requests) != 1 {
		t.Fatalf("pending after=%+v", pendingAfter.Requests)
	}
	pendingID := pendingAfter.Requests[0].ID
	request[any](t, router, "POST", base+"/end", host.Token, nil, 200)
	var state string
	if err := s.Repo.Pool.QueryRow(context.Background(), `SELECT status FROM mission_validations WHERE assignment_id=$1`, next.ID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "expired" {
		t.Fatalf("pending state=%s", state)
	}
	if call(router, "POST", base+"/validations/"+pendingID+"/resolve", guest.Token, map[string]bool{"approved": true}).Code != 409 {
		t.Fatal("validation after finish succeeded")
	}
}

func TestChaosThresholdProgressesWithTime(t *testing.T) {
	s, router := flowRouter(t)
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return now }
	host := request[models.Session](t, router, "POST", "/api/boxes/PB001/games", "", map[string]any{"name": "Chaos", "player_name": "Host", "mode": "chaos", "duration_minutes": 15}, 201)
	base := "/api/games/" + host.Game.ID
	request[models.Session](t, router, "POST", base+"/join", "", map[string]string{"name": "Guest"}, 201)
	request[any](t, router, "POST", base+"/start", host.Token, nil, 200)
	for _, step := range []struct {
		advance time.Duration
		want    int
	}{{0, 3}, {11 * time.Minute, 2}, {3*time.Minute + 1*time.Second, 1}} {
		now = now.Add(step.advance)
		body := call(router, "GET", base+"/chaos", host.Token, nil)
		if body.Code != 200 {
			t.Fatalf("chaos %d: %s", body.Code, body.Body.String())
		}
		var state struct {
			Progress struct {
				TriggerAt int `json:"trigger_at"`
			} `json:"progress"`
		}
		if err := json.Unmarshal(body.Body.Bytes(), &state); err != nil {
			t.Fatal(err)
		}
		if state.Progress.TriggerAt != step.want {
			t.Fatalf("threshold=%d want %d", state.Progress.TriggerAt, step.want)
		}
	}
	mission := flowMission(t, router, host.Token)
	request[models.Completion](t, router, "POST", "/api/players/me/mission/complete", host.Token, map[string]any{"assignment_id": mission.ID}, 200)
	state := call(router, "GET", base+"/chaos", host.Token, nil)
	var after struct {
		Sequence int `json:"sequence"`
	}
	if err := json.Unmarshal(state.Body.Bytes(), &after); err != nil {
		t.Fatal(err)
	}
	if after.Sequence != 1 {
		t.Fatalf("last-minute completion did not trigger Chaos: %s", state.Body.String())
	}
}
