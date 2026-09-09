package handlers_test

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"partybox/backend/internal/handlers"
	mlranker "partybox/backend/internal/ml"
	"partybox/backend/internal/models"
	"partybox/backend/internal/realtime"
	"partybox/backend/internal/repositories"
	"partybox/backend/internal/services"
)

type testMissionScorer struct{}

func (testMissionScorer) Score(input mlranker.MissionContext) float64 {
	return float64(input.Points)
}

func TestOptionalMLRankingAssignsAndNeverRepeatsImmediately(t *testing.T) {
	pool, dir := isolatedDB(t)
	migrateAndSeed(t, pool, dir)
	gin.SetMode(gin.TestMode)
	service := &services.Service{Repo: &repositories.Repository{Pool: pool, MissionScorer: testMissionScorer{}}}
	realtimeServer := realtime.NewServer("http://localhost:3000", service.AuthorizeRealtime)
	t.Cleanup(realtimeServer.Close)
	router := handlers.Router(service, realtimeServer, "http://localhost:3000")

	host := request[models.Session](t, router, http.MethodPost, "/api/boxes/PB001/games", "", map[string]any{
		"name": "ML ranking", "player_name": "Host", "mode": models.ModeSecretMissions,
	}, http.StatusCreated)
	request[models.Session](t, router, http.MethodPost, "/api/games/"+host.Game.ID+"/join", "",
		map[string]string{"name": "Guest"}, http.StatusCreated)
	request[any](t, router, http.MethodPost, "/api/games/"+host.Game.ID+"/start", host.Token, nil, http.StatusOK)
	mission := missionFor(t, router, host)
	seen := map[int]bool{}
	for range 10 {
		if seen[mission.MissionID] {
			t.Fatalf("ranked assignment repeated mission %d while unseen missions remain", mission.MissionID)
		}
		seen[mission.MissionID] = true
		completion := request[models.Completion](t, router, http.MethodPost, "/api/players/me/mission/complete", host.Token,
			map[string]string{"assignment_id": mission.ID}, http.StatusOK)
		if completion.Mission == nil {
			t.Fatal("ranked assignment did not produce a new mission")
		}
		mission = *completion.Mission
	}
}
