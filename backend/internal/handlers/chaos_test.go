package handlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	chaosgame "partybox/backend/internal/chaos"
	"partybox/backend/internal/handlers"
	"partybox/backend/internal/models"
	"partybox/backend/internal/realtime"
	"partybox/backend/internal/repositories"
	"partybox/backend/internal/services"
)

type scriptedChaosSelector struct {
	events     []chaosgame.EventType
	targetName string
}

func (selector *scriptedChaosSelector) ChooseEvent(options []chaosgame.EventType) chaosgame.EventType {
	if len(selector.events) == 0 {
		return options[0]
	}
	selected := selector.events[0]
	selector.events = selector.events[1:]
	return selected
}

func (selector *scriptedChaosSelector) ChoosePlayer(players []models.Player) models.Player {
	for _, player := range players {
		if player.Name == selector.targetName {
			return player
		}
	}
	return players[0]
}

func newChaosApp(t *testing.T, selector chaosgame.Selector) (*pgxpool.Pool, http.Handler) {
	t.Helper()
	pool, dir := isolatedDB(t)
	migrateAndSeed(t, pool, dir)
	gin.SetMode(gin.TestMode)
	service := &services.Service{
		Repo:  &repositories.Repository{Pool: pool},
		Chaos: chaosgame.NewEngine(selector),
	}
	realtimeServer := realtime.NewServer("http://localhost:3000", service.AuthorizeRealtime)
	t.Cleanup(realtimeServer.Close)
	return pool, handlers.Router(service, realtimeServer, "http://localhost:3000")
}

func createStartedChaosGame(t *testing.T, router http.Handler) (models.Session, models.Session) {
	t.Helper()
	host := request[models.Session](t, router, http.MethodPost, "/api/boxes/PB001/games", "", map[string]any{
		"name": "Chaos test", "player_name": "Host", "mode": models.ModeChaos,
	}, http.StatusCreated)
	guest := request[models.Session](t, router, http.MethodPost, "/api/games/"+host.Game.ID+"/join", "",
		map[string]string{"name": "Guest"}, http.StatusCreated)
	request[any](t, router, http.MethodPost, "/api/games/"+host.Game.ID+"/start", host.Token, nil, http.StatusOK)
	return host, guest
}

func missionFor(t *testing.T, router http.Handler, session models.Session) models.Mission {
	t.Helper()
	return request[struct{ Mission models.Mission }](
		t, router, http.MethodGet, "/api/players/me/mission", session.Token, nil, http.StatusOK,
	).Mission
}

func completeFor(t *testing.T, router http.Handler, session models.Session) (models.Mission, models.Completion) {
	t.Helper()
	mission := missionFor(t, router, session)
	completion := request[models.Completion](
		t, router, http.MethodPost, "/api/players/me/mission/complete", session.Token,
		map[string]string{"assignment_id": mission.ID}, http.StatusOK,
	)
	return mission, completion
}

func chaosStateFor(t *testing.T, router http.Handler, session models.Session) chaosgame.Snapshot {
	t.Helper()
	return request[chaosgame.Snapshot](
		t, router, http.MethodGet, "/api/games/"+session.Game.ID+"/chaos", session.Token, nil, http.StatusOK,
	)
}

func triggerChaosEvent(t *testing.T, router http.Handler, host models.Session) {
	t.Helper()
	for range chaosgame.CompletionThreshold {
		mission, completion := completeFor(t, router, host)
		if completion.AwardedPoints != mission.Points {
			t.Fatalf("pre-event completion awarded=%d, want %d", completion.AwardedPoints, mission.Points)
		}
	}
}

func TestChaosModeLifecycleAndIsolation(t *testing.T) {
	pool, router := newChaosApp(t, &scriptedChaosSelector{events: []chaosgame.EventType{chaosgame.EventDoubleTrouble}})
	host, guest := createStartedChaosGame(t, router)
	if host.Game.Mode != models.ModeChaos || guest.Game.Mode != models.ModeChaos {
		t.Fatal("Chaos mode was not preserved across create and join")
	}
	for _, session := range []models.Session{host, guest} {
		mission := missionFor(t, router, session)
		assertMissionMode(t, pool, mission.MissionID, models.ModeChaos)
	}
	state := chaosStateFor(t, router, host)
	if state.Active || state.Event != nil || state.Progress.Completed != 0 || state.Progress.TriggerAt != chaosgame.CompletionThreshold {
		t.Fatalf("unexpected initial Chaos state: %+v", state)
	}
	request[any](t, router, http.MethodPost, "/api/games/"+host.Game.ID+"/end", host.Token, nil, http.StatusOK)

	for _, mode := range []models.GameMode{models.ModeSecretMissions, models.ModeTreasureHunt} {
		session := request[models.Session](t, router, http.MethodPost, "/api/boxes/PB001/games", "", map[string]any{
			"name": string(mode), "player_name": "Host " + string(mode), "mode": mode,
		}, http.StatusCreated)
		joined := request[models.Session](t, router, http.MethodPost, "/api/games/"+session.Game.ID+"/join", "",
			map[string]string{"name": "Guest " + string(mode)}, http.StatusCreated)
		request[any](t, router, http.MethodPost, "/api/games/"+session.Game.ID+"/start", session.Token, nil, http.StatusOK)
		mission, completion := completeFor(t, router, session)
		if completion.AwardedPoints != mission.Points {
			t.Fatalf("%s received Chaos scoring", mode)
		}
		request[any](t, router, http.MethodGet, "/api/games/"+session.Game.ID+"/chaos", joined.Token, nil, http.StatusConflict)
		var states int
		if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM chaos_states WHERE game_id=$1`, session.Game.ID).Scan(&states); err != nil {
			t.Fatal(err)
		}
		if states != 0 {
			t.Fatalf("%s created Chaos runtime state", mode)
		}
		request[any](t, router, http.MethodPost, "/api/games/"+session.Game.ID+"/end", session.Token, nil, http.StatusOK)
	}
}

func TestChaosDoubleTrouble(t *testing.T) {
	pool, router := newChaosApp(t, &scriptedChaosSelector{events: []chaosgame.EventType{chaosgame.EventDoubleTrouble}})
	host, _ := createStartedChaosGame(t, router)
	triggerChaosEvent(t, router, host)
	state := chaosStateFor(t, router, host)
	if !state.Active || state.Event == nil || state.Event.Type != chaosgame.EventDoubleTrouble ||
		state.Event.RemainingUses != chaosgame.DoubleTroubleUses || state.Sequence != 1 {
		t.Fatalf("Double Trouble was not triggered: %+v", state)
	}

	firstMission, first := completeFor(t, router, host)
	if first.AwardedPoints != firstMission.Points*chaosgame.DoubleMultiplier {
		t.Fatalf("Double Trouble awarded=%d, want %d", first.AwardedPoints, firstMission.Points*2)
	}
	retry := request[models.Completion](t, router, http.MethodPost, "/api/players/me/mission/complete", host.Token,
		map[string]string{"assignment_id": firstMission.ID}, http.StatusOK)
	if !retry.AlreadyCompleted || retry.AwardedPoints != 0 {
		t.Fatalf("retry changed score: %+v", retry)
	}
	state = chaosStateFor(t, router, host)
	if state.Event == nil || state.Event.RemainingUses != 2 {
		t.Fatalf("retry consumed Double Trouble: %+v", state)
	}

	lastAwarded := first.AwardedPoints
	for wantRemaining := 1; wantRemaining >= 0; wantRemaining-- {
		mission, completion := completeFor(t, router, host)
		if completion.AwardedPoints != mission.Points*chaosgame.DoubleMultiplier {
			t.Fatal("Double Trouble did not double an awarded score")
		}
		lastAwarded = completion.AwardedPoints
		state = chaosStateFor(t, router, host)
		if wantRemaining == 0 {
			if state.Active || state.Event != nil {
				t.Fatalf("Double Trouble still active after three uses: %+v", state)
			}
		} else if state.Event == nil || state.Event.RemainingUses != wantRemaining {
			t.Fatalf("remaining uses=%+v, want %d", state.Event, wantRemaining)
		}
	}

	var triggered, consumed int
	if err := pool.QueryRow(context.Background(), `SELECT
		count(*) FILTER (WHERE type='chaos_event_triggered'),
		count(*) FILTER (WHERE type='chaos_event_consumed')
		FROM game_events WHERE game_id=$1`, host.Game.ID).Scan(&triggered, &consumed); err != nil {
		t.Fatal(err)
	}
	if triggered != 1 || consumed != 3 {
		t.Fatalf("Double Trouble events: triggered=%d consumed=%d", triggered, consumed)
	}
	var storedPoints int
	if err := pool.QueryRow(context.Background(), `SELECT (payload->>'points')::int FROM game_events
		WHERE game_id=$1 AND type='mission_completed' ORDER BY created_at DESC,id DESC LIMIT 1`, host.Game.ID).Scan(&storedPoints); err != nil {
		t.Fatal(err)
	}
	if storedPoints != lastAwarded {
		t.Fatalf("stored points=%d, want final award %d", storedPoints, lastAwarded)
	}
}

func TestChaosBounty(t *testing.T) {
	pool, router := newChaosApp(t, &scriptedChaosSelector{
		events: []chaosgame.EventType{chaosgame.EventBounty}, targetName: "Guest",
	})
	host, guest := createStartedChaosGame(t, router)
	triggerChaosEvent(t, router, host)
	state := chaosStateFor(t, router, host)
	if state.Event == nil || state.Event.Type != chaosgame.EventBounty ||
		state.Event.TargetPlayerID == nil || *state.Event.TargetPlayerID != guest.Player.ID ||
		state.Event.TargetPlayerName == nil || *state.Event.TargetPlayerName != guest.Player.Name ||
		state.Event.Bonus != chaosgame.BountyBonus {
		t.Fatalf("invalid bounty target: %+v", state)
	}

	hostMission, hostCompletion := completeFor(t, router, host)
	if hostCompletion.AwardedPoints != hostMission.Points {
		t.Fatal("a non-target player received the bounty")
	}
	state = chaosStateFor(t, router, host)
	if state.Event == nil || state.Event.RemainingUses != 1 {
		t.Fatalf("a non-target player consumed the bounty: %+v", state)
	}

	guestMission, guestCompletion := completeFor(t, router, guest)
	if guestCompletion.AwardedPoints != guestMission.Points+chaosgame.BountyBonus {
		t.Fatalf("bounty awarded=%d, want %d", guestCompletion.AwardedPoints, guestMission.Points+100)
	}
	retry := request[models.Completion](t, router, http.MethodPost, "/api/players/me/mission/complete", guest.Token,
		map[string]string{"assignment_id": guestMission.ID}, http.StatusOK)
	if !retry.AlreadyCompleted || retry.AwardedPoints != 0 {
		t.Fatalf("bounty retry was not idempotent: %+v", retry)
	}
	state = chaosStateFor(t, router, host)
	if state.Active || state.Event != nil {
		t.Fatalf("consumed bounty is still active: %+v", state)
	}

	var consumed int
	var consumer string
	if err := pool.QueryRow(context.Background(), `SELECT count(*),max(player_id::text)
		FROM game_events WHERE game_id=$1 AND type='chaos_event_consumed'`, host.Game.ID).Scan(&consumed, &consumer); err != nil {
		t.Fatal(err)
	}
	if consumed != 1 || consumer != guest.Player.ID {
		t.Fatalf("bounty consumption count=%d player=%s", consumed, consumer)
	}
}

func TestChaosMissionShuffle(t *testing.T) {
	pool, router := newChaosApp(t, &scriptedChaosSelector{events: []chaosgame.EventType{chaosgame.EventMissionShuffle}})
	host, guest := createStartedChaosGame(t, router)
	for range chaosgame.CompletionThreshold - 1 {
		completeFor(t, router, host)
	}
	guestBefore := missionFor(t, router, guest)
	hostBefore := missionFor(t, router, host)
	playerBefore := request[models.Player](t, router, http.MethodGet, "/api/players/me", host.Token, nil, http.StatusOK)
	completion := request[models.Completion](t, router, http.MethodPost, "/api/players/me/mission/complete", host.Token,
		map[string]string{"assignment_id": hostBefore.ID}, http.StatusOK)
	if completion.AwardedPoints != hostBefore.Points || completion.Player.Score != playerBefore.Score+hostBefore.Points {
		t.Fatalf("Mission Shuffle changed awarded points: %+v", completion)
	}
	if completion.Player.CompletedMissions != chaosgame.CompletionThreshold {
		t.Fatalf("cancelled assignments changed completed count: %+v", completion.Player)
	}
	state := chaosStateFor(t, router, host)
	if !state.Active || state.Event == nil || state.Event.Type != chaosgame.EventMissionShuffle || state.Event.RemainingUses != 0 {
		t.Fatalf("Mission Shuffle was not exposed: %+v", state)
	}

	guestAfter := missionFor(t, router, guest)
	if guestAfter.ID == guestBefore.ID || completion.Mission == nil {
		t.Fatal("Mission Shuffle did not replace every current mission")
	}
	for _, mission := range []models.Mission{guestAfter, *completion.Mission} {
		assertMissionMode(t, pool, mission.MissionID, models.ModeChaos)
	}
	var assigned, cancelled, playersWithOne int
	if err := pool.QueryRow(context.Background(), `SELECT
		count(*) FILTER (WHERE pm.status='assigned'),
		count(*) FILTER (WHERE pm.status='cancelled'),
		count(DISTINCT pm.player_id) FILTER (WHERE pm.status='assigned')
		FROM player_missions pm JOIN players p ON p.id=pm.player_id WHERE p.game_id=$1`, host.Game.ID).
		Scan(&assigned, &cancelled, &playersWithOne); err != nil {
		t.Fatal(err)
	}
	if assigned != 2 || cancelled != 2 || playersWithOne != 2 {
		t.Fatalf("shuffle assignments: assigned=%d cancelled=%d players=%d", assigned, cancelled, playersWithOne)
	}
	var guestOldStatus string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM player_missions WHERE id=$1`, guestBefore.ID).Scan(&guestOldStatus); err != nil {
		t.Fatal(err)
	}
	if guestOldStatus != "cancelled" {
		t.Fatalf("old guest mission status=%s", guestOldStatus)
	}
	request[any](t, router, http.MethodPost, "/api/players/me/mission/complete", guest.Token,
		map[string]string{"assignment_id": guestBefore.ID}, http.StatusConflict)

	var triggered, consumed, cancelledEvents, cancelledPlayers int
	if err := pool.QueryRow(context.Background(), `SELECT
		count(*) FILTER (WHERE type='chaos_event_triggered'),
		count(*) FILTER (WHERE type='chaos_event_consumed'),
		count(*) FILTER (WHERE type='mission_cancelled'),
		count(DISTINCT player_id) FILTER (WHERE type='mission_cancelled')
		FROM game_events WHERE game_id=$1`, host.Game.ID).
		Scan(&triggered, &consumed, &cancelledEvents, &cancelledPlayers); err != nil {
		t.Fatal(err)
	}
	if triggered != 1 || consumed != 1 || cancelledEvents != 2 || cancelledPlayers != 2 {
		t.Fatalf("shuffle events: triggered=%d consumed=%d cancelled=%d players=%d",
			triggered, consumed, cancelledEvents, cancelledPlayers)
	}
}
