package handlers_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"partybox/backend/internal/models"
)

func TestTreasureMissionSkip(t *testing.T) {
	service, router := flowRouter(t)
	host := request[models.Session](t, router, "POST", "/api/boxes/PB001/games", "", map[string]any{
		"name": "Chasse", "player_name": "Host", "mode": "treasure_hunt", "validation_mode": "peer", "duration_minutes": 30,
	}, 201)
	base := "/api/games/" + host.Game.ID
	guest := request[models.Session](t, router, "POST", base+"/join", "", map[string]string{"name": "Guest"}, 201)
	request[any](t, router, "POST", base+"/start", host.Token, nil, 200)
	mission := flowMission(t, router, host.Token)
	path := "/api/players/me/mission/skip"
	body := map[string]string{"assignment_id": mission.ID}
	if got := call(router, "POST", path, guest.Token, body); got.Code != 404 {
		t.Fatalf("other player's assignment: %d %s", got.Code, got.Body.String())
	}
	if got := call(router, "POST", path, "", body); got.Code != 401 {
		t.Fatalf("unauthenticated skip: %d", got.Code)
	}
	request[models.Completion](t, router, "POST", "/api/players/me/mission/complete", host.Token, body, 200)
	result := request[models.MissionSkip](t, router, "POST", path, host.Token, body, 200)
	if result.Mission.ID == mission.ID || result.Mission.MissionID == mission.MissionID || result.Player.Score != 0 || result.Player.CompletedMissions != 0 {
		t.Fatalf("skipped hunt: %+v", result)
	}
	if got := flowMission(t, router, host.Token); got.ID != result.Mission.ID {
		t.Fatalf("current hunt: %+v", got)
	}
	var assignmentStatus, validationStatus string
	if err := service.Repo.Pool.QueryRow(context.Background(), `SELECT pm.status,v.status FROM player_missions pm JOIN mission_validations v ON v.assignment_id=pm.id WHERE pm.id=$1`, mission.ID).Scan(&assignmentStatus, &validationStatus); err != nil {
		t.Fatal(err)
	}
	if assignmentStatus != "cancelled" || validationStatus != "expired" {
		t.Fatalf("assignment=%s validation=%s", assignmentStatus, validationStatus)
	}
	if got := call(router, "POST", path, host.Token, body); got.Code != 409 {
		t.Fatalf("duplicate skip: %d", got.Code)
	}
	if got := request[models.Player](t, router, "GET", "/api/players/me", host.Token, nil, http.StatusOK); got.Score != 0 {
		t.Fatalf("skip added points: %+v", got)
	}
	request[any](t, router, "POST", base+"/end", host.Token, nil, 200)
	if got := call(router, "POST", path, host.Token, map[string]string{"assignment_id": result.Mission.ID}); got.Code != 409 {
		t.Fatalf("skip after finish: %d", got.Code)
	}
}

func TestTreasureMissionSkipRejectsOtherModesAndGames(t *testing.T) {
	service, router := flowRouter(t)
	host, _, base := flowGame(t, router, "PB001", 30, "visible", "trust")
	mission := flowMission(t, router, host.Token)
	path := "/api/players/me/mission/skip"
	if got := call(router, "POST", path, host.Token, map[string]string{"assignment_id": mission.ID}); got.Code != 409 {
		t.Fatalf("secret mission skipped: %d", got.Code)
	}
	request[any](t, router, "POST", base+"/end", host.Token, nil, 200)
	mustExec(t, service.Repo.Pool, `INSERT INTO boxes(id,name) VALUES('PB903','Other')`)
	other := request[models.Session](t, router, "POST", "/api/boxes/PB903/games", "", map[string]any{
		"name": "Other hunt", "player_name": "Other host", "mode": "treasure_hunt", "duration_minutes": 30,
	}, 201)
	otherBase := "/api/games/" + other.Game.ID
	request[models.Session](t, router, "POST", otherBase+"/join", "", map[string]string{"name": "Other guest"}, 201)
	request[any](t, router, "POST", otherBase+"/start", other.Token, nil, 200)
	if got := call(router, "POST", path, other.Token, map[string]string{"assignment_id": mission.ID}); got.Code != 404 {
		t.Fatalf("cross-game assignment: %d", got.Code)
	}
}

func TestTreasureMissionSkipAtDeadline(t *testing.T) {
	service, router := flowRouter(t)
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	host := request[models.Session](t, router, "POST", "/api/boxes/PB001/games", "", map[string]any{
		"name": "Timed hunt", "player_name": "Host", "mode": "treasure_hunt", "duration_minutes": 15,
	}, 201)
	base := "/api/games/" + host.Game.ID
	request[models.Session](t, router, "POST", base+"/join", "", map[string]string{"name": "Guest"}, 201)
	request[any](t, router, "POST", base+"/start", host.Token, nil, 200)
	mission := flowMission(t, router, host.Token)
	now = now.Add(15 * time.Minute)
	if got := call(router, "POST", "/api/players/me/mission/skip", host.Token, map[string]string{"assignment_id": mission.ID}); got.Code != 409 {
		t.Fatalf("skip at deadline: %d %s", got.Code, got.Body.String())
	}
	if got := request[models.Game](t, router, "GET", base, host.Token, nil, 200); got.Status != "ended" {
		t.Fatalf("game after deadline: %+v", got)
	}
}
