package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"partybox/backend/internal/ai"
	"partybox/backend/internal/handlers"
	"partybox/backend/internal/models"
	"partybox/backend/internal/realtime"
	"partybox/backend/internal/repositories"
	"partybox/backend/internal/services"
)

type fakeMissionGenerator struct {
	generateFn func(ctx context.Context, input ai.GenerationRequest) ([]ai.GeneratedMission, error)
}

func (f *fakeMissionGenerator) Generate(ctx context.Context, input ai.GenerationRequest) ([]ai.GeneratedMission, error) {
	if f.generateFn != nil {
		return f.generateFn(ctx, input)
	}
	count := input.Count
	if count <= 0 {
		count = 20
	}
	missions := make([]ai.GeneratedMission, count)
	for i := range missions {
		missions[i] = ai.GeneratedMission{
			Text:       fmt.Sprintf("Mission IA %s #%d pour vibe %s", input.Mode, i+1, input.Vibe),
			Category:   "ia_custom",
			Difficulty: 2,
			Points:     20,
		}
	}
	return missions, nil
}

func TestAIMissionGenerationSuite(t *testing.T) {
	pool, dir := isolatedDB(t)
	migrateAndSeed(t, pool, dir)
	gin.SetMode(gin.TestMode)

	fakeGen := &fakeMissionGenerator{}
	service := &services.Service{
		Repo: &repositories.Repository{Pool: pool},
		AI:   fakeGen,
	}
	realtimeServer := realtime.NewServer("http://localhost:3000", service.AuthorizeRealtime)
	t.Cleanup(realtimeServer.Close)
	router := handlers.Router(service, realtimeServer, "http://localhost:3000")

	// 1. Create a game for testing
	hostA := request[models.Session](t, router, http.MethodPost, "/api/boxes/PB001/games", "", map[string]any{
		"name": "Game A", "player_name": "HostA", "mode": models.ModeSecretMissions,
	}, http.StatusCreated)
	baseA := "/api/games/" + hostA.Game.ID

	// Join guest
	guestA := request[models.Session](t, router, http.MethodPost, baseA+"/join", "", map[string]string{
		"name": "GuestA",
	}, http.StatusCreated)

	t.Run("status_before_generation", func(t *testing.T) {
		status := request[services.AIStatus](t, router, http.MethodGet, baseA+"/ai-missions/status", hostA.Token, nil, http.StatusOK)
		if !status.Available || status.Count != 0 {
			t.Fatalf("expected available=true, count=0, got %+v", status)
		}
		if status.RemainingGenerations != 5 {
			t.Fatalf("expected five available generations, got %+v", status)
		}
	})

	t.Run("non_host_rejected", func(t *testing.T) {
		request[any](t, router, http.MethodPost, baseA+"/ai-missions/generate", guestA.Token, map[string]any{
			"vibe": "fun", "intensity": 7, "count": 20,
		}, http.StatusForbidden)
	})

	t.Run("invalid_inputs_rejected", func(t *testing.T) {
		// Invalid vibe
		request[any](t, router, http.MethodPost, baseA+"/ai-missions/generate", hostA.Token, map[string]any{
			"vibe": "unknown_vibe", "intensity": 7, "count": 20,
		}, http.StatusBadRequest)

		// Invalid intensity
		request[any](t, router, http.MethodPost, baseA+"/ai-missions/generate", hostA.Token, map[string]any{
			"vibe": "fun", "intensity": 15, "count": 20,
		}, http.StatusBadRequest)
		request[any](t, router, http.MethodPost, baseA+"/ai-missions/generate", hostA.Token, map[string]any{
			"vibe": "fun", "intensity": -1, "count": 20,
		}, http.StatusBadRequest)

		// Context too long (> 300 chars)
		request[any](t, router, http.MethodPost, baseA+"/ai-missions/generate", hostA.Token, map[string]any{
			"vibe": "fun", "intensity": 5, "context": strings.Repeat("A", 301), "count": 20,
		}, http.StatusBadRequest)

		// Count out of bounds (< 5 or > 50)
		request[any](t, router, http.MethodPost, baseA+"/ai-missions/generate", hostA.Token, map[string]any{
			"vibe": "fun", "intensity": 5, "count": 2,
		}, http.StatusBadRequest)
		request[any](t, router, http.MethodPost, baseA+"/ai-missions/generate", hostA.Token, map[string]any{
			"vibe": "fun", "intensity": 5, "count": 55,
		}, http.StatusBadRequest)
	})

	t.Run("invalid_ai_response_rejected_and_not_persisted", func(t *testing.T) {
		// Generator returns invalid mission (negative points)
		fakeGen.generateFn = func(ctx context.Context, input ai.GenerationRequest) ([]ai.GeneratedMission, error) {
			return []ai.GeneratedMission{
				{Text: "Invalid", Category: "test", Difficulty: 1, Points: -10},
			}, nil
		}
		defer func() { fakeGen.generateFn = nil }()

		request[any](t, router, http.MethodPost, baseA+"/ai-missions/generate", hostA.Token, map[string]any{
			"vibe": "fun", "intensity": 5, "count": 20,
		}, http.StatusBadRequest)

		var count int
		if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM missions WHERE game_id=$1 AND source='ai'`, hostA.Game.ID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("expected 0 missions in DB, found %d", count)
		}
	})

	t.Run("host_generates_missions_successfully", func(t *testing.T) {
		res := request[map[string]int](t, router, http.MethodPost, baseA+"/ai-missions/generate", hostA.Token, map[string]any{
			"vibe": "fun", "intensity": 7, "context": "Soirée en appartement", "count": 20,
		}, http.StatusOK)
		if res["generated"] != 20 {
			t.Fatalf("expected 20 generated, got %d", res["generated"])
		}

		status := request[services.AIStatus](t, router, http.MethodGet, baseA+"/ai-missions/status", hostA.Token, nil, http.StatusOK)
		if !status.Available || status.Count != 20 {
			t.Fatalf("expected available=true, count=20, got %+v", status)
		}

		// Verify missions stored in DB with source='ai' and game_id=hostA.Game.ID
		var count int
		if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM missions WHERE game_id=$1 AND source='ai'`, hostA.Game.ID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 20 {
			t.Fatalf("expected 20 stored missions in DB, got %d", count)
		}

		// Verify game_event ai_missions_generated was recorded
		var eventCount int
		var payload json.RawMessage
		if err := pool.QueryRow(context.Background(), `SELECT count(*), payload FROM game_events WHERE game_id=$1 AND type='ai_missions_generated' GROUP BY payload`, hostA.Game.ID).Scan(&eventCount, &payload); err != nil {
			t.Fatal(err)
		}
		if eventCount != 1 {
			t.Fatalf("expected 1 ai_missions_generated event, got %d", eventCount)
		}
		var evtData map[string]any
		_ = json.Unmarshal(payload, &evtData)
		if evtData["vibe"] != "fun" || evtData["mode"] != string(models.ModeSecretMissions) {
			t.Fatalf("unexpected event payload: %s", string(payload))
		}
	})

	t.Run("provider_failure_preserves_existing_catalog", func(t *testing.T) {
		fakeGen.generateFn = func(ctx context.Context, input ai.GenerationRequest) ([]ai.GeneratedMission, error) {
			return nil, errors.New("OpenAI timeout simulation")
		}
		defer func() { fakeGen.generateFn = nil }()

		request[any](t, router, http.MethodPost, baseA+"/ai-missions/generate", hostA.Token, map[string]any{
			"vibe": "chaos", "intensity": 9, "count": 20,
		}, http.StatusInternalServerError)

		// Check that the existing 20 missions were preserved
		var count int
		if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM missions WHERE game_id=$1 AND source='ai'`, hostA.Game.ID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 20 {
			t.Fatalf("expected 20 missions to remain after provider error, got %d", count)
		}
	})

	t.Run("regeneration_atomically_replaces_catalog", func(t *testing.T) {
		fakeGen.generateFn = func(ctx context.Context, input ai.GenerationRequest) ([]ai.GeneratedMission, error) {
			missions := make([]ai.GeneratedMission, 15)
			for i := range missions {
				missions[i] = ai.GeneratedMission{
					Text:       fmt.Sprintf("Regenerated mission #%d", i+1),
					Category:   "regenerated",
					Difficulty: 1,
					Points:     15,
				}
			}
			return missions, nil
		}
		defer func() { fakeGen.generateFn = nil }()

		res := request[map[string]int](t, router, http.MethodPost, baseA+"/ai-missions/generate", hostA.Token, map[string]any{
			"vibe": "chill", "intensity": 3, "count": 15,
		}, http.StatusOK)
		if res["generated"] != 15 {
			t.Fatalf("expected 15 regenerated, got %d", res["generated"])
		}

		var count int
		if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM missions WHERE game_id=$1 AND source='ai'`, hostA.Game.ID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 15 {
			t.Fatalf("expected exactly 15 missions after replacement, got %d", count)
		}
	})

	t.Run("game_isolation_and_ai_priority_vs_seed_fallback", func(t *testing.T) {
		mustExec(t, pool, "INSERT INTO boxes(id, name) VALUES ('PB002', 'Deuxième Box') ON CONFLICT DO NOTHING")
		// Game A has 15 AI missions.
		// Create Game B without AI missions (using seed fallback) on PB002.
		hostB := request[models.Session](t, router, http.MethodPost, "/api/boxes/PB002/games", "", map[string]any{
			"name": "Game B (Seed)", "player_name": "HostB", "mode": models.ModeSecretMissions,
		}, http.StatusCreated)
		baseB := "/api/games/" + hostB.Game.ID
		guestB := request[models.Session](t, router, http.MethodPost, baseB+"/join", "", map[string]string{
			"name": "GuestB",
		}, http.StatusCreated)

		// Start Game A (with AI missions)
		request[any](t, router, http.MethodPost, baseA+"/start", hostA.Token, nil, http.StatusOK)
		// Start Game B (without AI missions)
		request[any](t, router, http.MethodPost, baseB+"/start", hostB.Token, nil, http.StatusOK)

		// Check Game A assignment: MUST be one of the AI missions from Game A
		missionA := request[struct{ Mission models.Mission }](t, router, http.MethodGet, "/api/players/me/mission", hostA.Token, nil, http.StatusOK).Mission
		if !strings.HasPrefix(missionA.Text, "Regenerated mission #") {
			t.Fatalf("Game A should receive custom AI mission, got: %s", missionA.Text)
		}

		// Check Game B assignment: MUST be a seed mission, NEVER a mission from Game A
		missionB := request[struct{ Mission models.Mission }](t, router, http.MethodGet, "/api/players/me/mission", hostB.Token, nil, http.StatusOK).Mission
		if strings.HasPrefix(missionB.Text, "Regenerated mission #") || strings.HasPrefix(missionB.Text, "Mission IA") {
			t.Fatalf("Game B leaked AI mission from Game A: %s", missionB.Text)
		}

		// Verify in DB that no mission from Game A was assigned to any player in Game B
		var leakCount int
		if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM player_missions pm
			JOIN missions m ON m.id = pm.mission_id
			JOIN players p ON p.id = pm.player_id
			WHERE p.game_id = $1 AND m.game_id = $2`, hostB.Game.ID, hostA.Game.ID).Scan(&leakCount); err != nil {
			t.Fatal(err)
		}
		if leakCount != 0 {
			t.Fatalf("cross-game leakage: %d assignments from Game A given to Game B", leakCount)
		}

		_ = guestB
	})

	t.Run("generation_rejected_after_game_start", func(t *testing.T) {
		// Game A is now "playing"
		request[any](t, router, http.MethodPost, baseA+"/ai-missions/generate", hostA.Token, map[string]any{
			"vibe": "fun", "intensity": 5, "count": 20,
		}, http.StatusConflict)
	})

	t.Run("treasure_hunt_ai_generation_also_supported", func(t *testing.T) {
		// End previous games on box to free it up if needed
		request[any](t, router, http.MethodPost, baseA+"/end", hostA.Token, nil, http.StatusOK)

		hostTH := request[models.Session](t, router, http.MethodPost, "/api/boxes/PB001/games", "", map[string]any{
			"name": "Treasure Hunt AI", "player_name": "ExplorerHost", "mode": models.ModeTreasureHunt,
		}, http.StatusCreated)
		baseTH := "/api/games/" + hostTH.Game.ID
		request[models.Session](t, router, http.MethodPost, baseTH+"/join", "", map[string]string{"name": "ExplorerGuest"}, http.StatusCreated)

		res := request[map[string]int](t, router, http.MethodPost, baseTH+"/ai-missions/generate", hostTH.Token, map[string]any{
			"vibe": "chill", "intensity": 4, "count": 20,
		}, http.StatusOK)
		if res["generated"] != 20 {
			t.Fatalf("expected 20 generated for treasure hunt, got %d", res["generated"])
		}

		request[any](t, router, http.MethodPost, baseTH+"/start", hostTH.Token, nil, http.StatusOK)
		thMission := request[struct{ Mission models.Mission }](t, router, http.MethodGet, "/api/players/me/mission", hostTH.Token, nil, http.StatusOK).Mission
		if !strings.Contains(thMission.Text, "treasure_hunt") {
			t.Fatalf("expected treasure_hunt AI mission, got: %s", thMission.Text)
		}
	})

	t.Run("generation_attempts_are_limited_per_game", func(t *testing.T) {
		mustExec(t, pool, "INSERT INTO boxes(id, name) VALUES ('PB003', 'Troisième Box') ON CONFLICT DO NOTHING")
		host := request[models.Session](t, router, http.MethodPost, "/api/boxes/PB003/games", "", map[string]any{
			"name": "Limited AI", "player_name": "LimitedHost", "mode": models.ModeSecretMissions,
		}, http.StatusCreated)
		base := "/api/games/" + host.Game.ID
		providerCalls := 0
		fakeGen.generateFn = func(ctx context.Context, input ai.GenerationRequest) ([]ai.GeneratedMission, error) {
			providerCalls++
			missions := make([]ai.GeneratedMission, input.Count)
			for i := range missions {
				missions[i] = ai.GeneratedMission{Text: fmt.Sprintf("Limited mission #%d", i+1), Category: "limited", Difficulty: 1, Points: 10}
			}
			return missions, nil
		}
		defer func() { fakeGen.generateFn = nil }()

		for range 5 {
			request[any](t, router, http.MethodPost, base+"/ai-missions/generate", host.Token, map[string]any{
				"vibe": "fun", "intensity": 5, "count": 5,
			}, http.StatusOK)
		}
		request[any](t, router, http.MethodPost, base+"/ai-missions/generate", host.Token, map[string]any{
			"vibe": "fun", "intensity": 5, "count": 5,
		}, http.StatusConflict)
		if providerCalls != 5 {
			t.Fatalf("provider called %d times, expected 5", providerCalls)
		}
		status := request[services.AIStatus](t, router, http.MethodGet, base+"/ai-missions/status", host.Token, nil, http.StatusOK)
		if status.RemainingGenerations != 0 || status.Count != 5 {
			t.Fatalf("unexpected limited status: %+v", status)
		}

		restartedService := &services.Service{Repo: &repositories.Repository{Pool: pool}, AI: fakeGen}
		restartedRealtime := realtime.NewServer("http://localhost:3000", restartedService.AuthorizeRealtime)
		defer restartedRealtime.Close()
		restartedRouter := handlers.Router(restartedService, restartedRealtime, "http://localhost:3000")
		restartedStatus := request[services.AIStatus](t, restartedRouter, http.MethodGet, base+"/ai-missions/status", host.Token, nil, http.StatusOK)
		if restartedStatus.RemainingGenerations != 0 {
			t.Fatalf("generation limit was lost after service restart: %+v", restartedStatus)
		}
		request[any](t, restartedRouter, http.MethodPost, base+"/ai-missions/generate", host.Token, map[string]any{
			"vibe": "fun", "intensity": 5, "count": 5,
		}, http.StatusConflict)
		if providerCalls != 5 {
			t.Fatalf("provider called after persisted limit: %d", providerCalls)
		}
	})

	t.Run("start_is_rejected_while_generation_is_running", func(t *testing.T) {
		mustExec(t, pool, "INSERT INTO boxes(id, name) VALUES ('PB004', 'Quatrième Box') ON CONFLICT DO NOTHING")
		host := request[models.Session](t, router, http.MethodPost, "/api/boxes/PB004/games", "", map[string]any{
			"name": "Concurrent AI", "player_name": "ConcurrentHost", "mode": models.ModeSecretMissions,
		}, http.StatusCreated)
		base := "/api/games/" + host.Game.ID
		request[models.Session](t, router, http.MethodPost, base+"/join", "", map[string]string{"name": "ConcurrentGuest"}, http.StatusCreated)

		started := make(chan struct{})
		release := make(chan struct{})
		var once sync.Once
		fakeGen.generateFn = func(ctx context.Context, input ai.GenerationRequest) ([]ai.GeneratedMission, error) {
			once.Do(func() { close(started) })
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			missions := make([]ai.GeneratedMission, input.Count)
			for i := range missions {
				missions[i] = ai.GeneratedMission{Text: fmt.Sprintf("Concurrent mission #%d", i+1), Category: "concurrent", Difficulty: 1, Points: 10}
			}
			return missions, nil
		}
		defer func() { fakeGen.generateFn = nil }()

		generationDone := make(chan int, 1)
		go func() {
			generationDone <- call(router, http.MethodPost, base+"/ai-missions/generate", host.Token, map[string]any{
				"vibe": "fun", "intensity": 5, "count": 5,
			}).Code
		}()
		<-started
		request[any](t, router, http.MethodPost, base+"/start", host.Token, nil, http.StatusConflict)
		close(release)
		if status := <-generationDone; status != http.StatusOK {
			t.Fatalf("generation returned %d, expected 200", status)
		}
		request[any](t, router, http.MethodPost, base+"/start", host.Token, nil, http.StatusOK)
	})
}
