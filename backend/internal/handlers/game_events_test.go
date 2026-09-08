package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"partybox/backend/internal/handlers"
	"partybox/backend/internal/models"
	"partybox/backend/internal/realtime"
	"partybox/backend/internal/repositories"
	"partybox/backend/internal/services"
)

func TestPersistedGameEventsFollowSuccessfulMutations(t *testing.T) {
	pool, dir := isolatedDB(t)
	migrateAndSeed(t, pool, dir)
	gin.SetMode(gin.TestMode)
	service := &services.Service{Repo: &repositories.Repository{Pool: pool}}
	realtimeServer := realtime.NewServer("http://localhost:3000", service.AuthorizeRealtime)
	t.Cleanup(realtimeServer.Close)
	router := handlers.Router(service, realtimeServer, "http://localhost:3000")

	host := request[models.Session](t, router, http.MethodPost, "/api/boxes/PB001/games", "", map[string]any{
		"name": "Event history", "player_name": "Host", "mode": models.ModeSecretMissions,
	}, http.StatusCreated)
	base := "/api/games/" + host.Game.ID
	guest := request[models.Session](t, router, http.MethodPost, base+"/join", "", map[string]string{
		"name": "Guest",
	}, http.StatusCreated)
	request[any](t, router, http.MethodPost, base+"/start", host.Token, nil, http.StatusOK)
	request[any](t, router, http.MethodPost, base+"/start", host.Token, nil, http.StatusOK)
	mission := request[struct{ Mission models.Mission }](
		t, router, http.MethodGet, "/api/players/me/mission", host.Token, nil, http.StatusOK,
	).Mission
	completion := request[models.Completion](t, router, http.MethodPost, "/api/players/me/mission/complete", host.Token,
		map[string]string{"assignment_id": mission.ID}, http.StatusOK)
	if completion.AwardedPoints != mission.Points {
		t.Fatalf("awarded=%d, want %d", completion.AwardedPoints, mission.Points)
	}
	duplicate := request[models.Completion](t, router, http.MethodPost, "/api/players/me/mission/complete", host.Token,
		map[string]string{"assignment_id": mission.ID}, http.StatusOK)
	if !duplicate.AlreadyCompleted {
		t.Fatal("completion retry was not idempotent")
	}
	request[any](t, router, http.MethodPost, base+"/end", host.Token, nil, http.StatusOK)
	request[any](t, router, http.MethodPost, base+"/end", host.Token, nil, http.StatusOK)

	rows, err := pool.Query(context.Background(), `SELECT type,player_id::text,payload
		FROM game_events WHERE game_id=$1 ORDER BY created_at,id`, host.Game.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	type storedEvent struct {
		typeName string
		playerID *string
		payload  map[string]any
	}
	var events []storedEvent
	for rows.Next() {
		var event storedEvent
		var payload []byte
		if err = rows.Scan(&event.typeName, &event.playerID, &payload); err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(payload, &event.payload); err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	wantTypes := []models.GameEventType{
		models.GameEventPlayerJoined,
		models.GameEventGameStarted,
		models.GameEventMissionCompleted,
		models.GameEventGameEnded,
	}
	if len(events) != len(wantTypes) {
		t.Fatalf("got %d events, want %d: %+v", len(events), len(wantTypes), events)
	}
	for index, wantType := range wantTypes {
		if events[index].typeName != string(wantType) {
			t.Fatalf("event %d type=%q, want %q", index, events[index].typeName, wantType)
		}
	}
	if events[0].playerID == nil || *events[0].playerID != guest.Player.ID {
		t.Fatalf("join player=%v, want %s", events[0].playerID, guest.Player.ID)
	}
	for _, index := range []int{1, 2, 3} {
		if events[index].playerID == nil || *events[index].playerID != host.Player.ID {
			t.Fatalf("event %d player=%v, want host %s", index, events[index].playerID, host.Player.ID)
		}
	}
	if events[2].payload["assignment_id"] != mission.ID ||
		int(events[2].payload["points"].(float64)) != mission.Points {
		t.Fatalf("completion payload=%v", events[2].payload)
	}
}
