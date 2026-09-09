package handlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"partybox/backend/internal/handlers"
	"partybox/backend/internal/models"
	"partybox/backend/internal/realtime"
	"partybox/backend/internal/repositories"
	"partybox/backend/internal/services"
)

func TestMissionFeedbackLifecycle(t *testing.T) {
	pool, dir := isolatedDB(t)
	migrateAndSeed(t, pool, dir)
	gin.SetMode(gin.TestMode)
	service := &services.Service{Repo: &repositories.Repository{Pool: pool}}
	realtimeServer := realtime.NewServer("http://localhost:3000", service.AuthorizeRealtime)
	t.Cleanup(realtimeServer.Close)
	router := handlers.Router(service, realtimeServer, "http://localhost:3000")

	host := request[models.Session](t, router, http.MethodPost, "/api/boxes/PB001/games", "", map[string]any{
		"name": "Feedback test", "player_name": "Host", "mode": models.ModeSecretMissions,
	}, http.StatusCreated)
	guest := request[models.Session](t, router, http.MethodPost, "/api/games/"+host.Game.ID+"/join", "",
		map[string]string{"name": "Guest"}, http.StatusCreated)
	request[any](t, router, http.MethodPost, "/api/games/"+host.Game.ID+"/start", host.Token, nil, http.StatusOK)

	active := missionFor(t, router, host)
	feedbackPath := func(assignmentID string) string {
		return "/api/players/me/missions/" + assignmentID + "/feedback"
	}
	request[any](t, router, http.MethodPut, feedbackPath(active.ID), host.Token,
		map[string]int{"rating": 1}, http.StatusConflict)
	request[any](t, router, http.MethodPut, feedbackPath(active.ID), host.Token,
		map[string]int{"rating": 2}, http.StatusBadRequest)
	request[any](t, router, http.MethodPut, feedbackPath(active.ID), host.Token,
		map[string]any{}, http.StatusBadRequest)

	var ratedAssignments []string
	for _, rating := range []int{-1, 0, 1} {
		mission, _ := completeFor(t, router, host)
		ratedAssignments = append(ratedAssignments, mission.ID)
		response := request[struct {
			AssignmentID string `json:"assignment_id"`
			Rating       int    `json:"rating"`
		}](t, router, http.MethodPut, feedbackPath(mission.ID), host.Token,
			map[string]int{"rating": rating}, http.StatusOK)
		if response.AssignmentID != mission.ID || response.Rating != rating {
			t.Fatalf("unexpected feedback response: %+v", response)
		}
	}

	updatedAssignment := ratedAssignments[len(ratedAssignments)-1]
	request[any](t, router, http.MethodPut, feedbackPath(updatedAssignment), host.Token,
		map[string]int{"rating": -1}, http.StatusOK)
	// An identical retry keeps both the row and the event stream idempotent.
	request[any](t, router, http.MethodPut, feedbackPath(updatedAssignment), host.Token,
		map[string]int{"rating": -1}, http.StatusOK)

	var rows, finalRating int
	if err := pool.QueryRow(context.Background(), `SELECT count(*),max(rating)
		FROM mission_feedback WHERE assignment_id=$1`, updatedAssignment).Scan(&rows, &finalRating); err != nil {
		t.Fatal(err)
	}
	if rows != 1 || finalRating != -1 {
		t.Fatalf("feedback rows=%d rating=%d, want one row rated -1", rows, finalRating)
	}

	guestCompleted, _ := completeFor(t, router, guest)
	request[any](t, router, http.MethodPut, feedbackPath(guestCompleted.ID), host.Token,
		map[string]int{"rating": 1}, http.StatusNotFound)

	guestActive := missionFor(t, router, guest)
	mustExec(t, pool, `UPDATE player_missions SET status='cancelled' WHERE id=$1`, guestActive.ID)
	request[any](t, router, http.MethodPut, feedbackPath(guestActive.ID), guest.Token,
		map[string]int{"rating": 0}, http.StatusConflict)

	var events, positiveUpdates, negativeUpdates int
	if err := pool.QueryRow(context.Background(), `SELECT count(*),
		count(*) FILTER (WHERE (payload->>'rating')::int=1),
		count(*) FILTER (WHERE (payload->>'rating')::int=-1)
		FROM game_events
		WHERE type='mission_feedback_submitted' AND payload->>'assignment_id'=$1`,
		updatedAssignment).Scan(&events, &positiveUpdates, &negativeUpdates); err != nil {
		t.Fatal(err)
	}
	if events != 2 || positiveUpdates != 1 || negativeUpdates != 1 {
		t.Fatalf("feedback events=%d positive=%d negative=%d", events, positiveUpdates, negativeUpdates)
	}

	var totalFeedback, totalEvents int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM mission_feedback`).Scan(&totalFeedback); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM game_events WHERE type='mission_feedback_submitted'`).Scan(&totalEvents); err != nil {
		t.Fatal(err)
	}
	if totalFeedback != 3 || totalEvents != 4 {
		t.Fatalf("feedback rows=%d events=%d, want 3 rows and 4 changes", totalFeedback, totalEvents)
	}
}

func TestMissionFeedbackWorksAcrossGameModes(t *testing.T) {
	for index, mode := range []models.GameMode{
		models.ModeSecretMissions,
		models.ModeTreasureHunt,
		models.ModeChaos,
	} {
		t.Run(string(mode), func(t *testing.T) {
			pool, dir := isolatedDB(t)
			migrateAndSeed(t, pool, dir)
			boxID := "PB-FEEDBACK-" + string(rune('A'+index))
			mustExec(t, pool, `INSERT INTO boxes(id,name) VALUES($1,'Feedback box')`, boxID)
			gin.SetMode(gin.TestMode)
			service := &services.Service{Repo: &repositories.Repository{Pool: pool}}
			realtimeServer := realtime.NewServer("http://localhost:3000", service.AuthorizeRealtime)
			t.Cleanup(realtimeServer.Close)
			router := handlers.Router(service, realtimeServer, "http://localhost:3000")

			host := request[models.Session](t, router, http.MethodPost, "/api/boxes/"+boxID+"/games", "", map[string]any{
				"name": "Mode feedback", "player_name": "Host", "mode": mode,
			}, http.StatusCreated)
			request[models.Session](t, router, http.MethodPost, "/api/games/"+host.Game.ID+"/join", "",
				map[string]string{"name": "Guest"}, http.StatusCreated)
			request[any](t, router, http.MethodPost, "/api/games/"+host.Game.ID+"/start", host.Token, nil, http.StatusOK)
			mission, _ := completeFor(t, router, host)
			request[any](t, router, http.MethodPut,
				"/api/players/me/missions/"+mission.ID+"/feedback", host.Token,
				map[string]int{"rating": 1}, http.StatusOK)

			var storedMode models.GameMode
			if err := pool.QueryRow(context.Background(), `SELECT g.mode
				FROM mission_feedback mf
				JOIN player_missions pm ON pm.id=mf.assignment_id
				JOIN players p ON p.id=pm.player_id
				JOIN games g ON g.id=p.game_id
				WHERE mf.assignment_id=$1`, mission.ID).Scan(&storedMode); err != nil {
				t.Fatal(err)
			}
			if storedMode != mode {
				t.Fatalf("stored feedback mode=%s, want %s", storedMode, mode)
			}
		})
	}
}
