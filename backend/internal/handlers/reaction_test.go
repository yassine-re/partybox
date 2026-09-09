package handlers_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"partybox/backend/internal/handlers"
	"partybox/backend/internal/models"
	"partybox/backend/internal/realtime"
	"partybox/backend/internal/repositories"
	"partybox/backend/internal/services"
)

type reactionApp struct {
	pool    *pgxpool.Pool
	router  http.Handler
	service *services.Service
	token   string
	now     *time.Time
}

func newReactionApp(t *testing.T) *reactionApp {
	t.Helper()
	pool, dir := isolatedDB(t)
	migrateAndSeed(t, pool, dir)
	current := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	service := &services.Service{
		Repo:       &repositories.Repository{Pool: pool},
		Reaction:   services.DefaultReactionConfig(),
		Now:        func() time.Time { return current },
		RandomIntN: func(int) int { return 0 },
	}
	token := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{42}, 32))
	if err := service.ProvisionDevice(context.Background(), "PB001", token); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	realtimeServer := realtime.NewServer("http://localhost:3000", service.AuthorizeRealtime)
	t.Cleanup(realtimeServer.Close)
	return &reactionApp{
		pool: pool, service: service, token: token, now: &current,
		router: handlers.Router(service, realtimeServer, "http://localhost:3000"),
	}
}

func (app *reactionApp) setNow(value time.Time) {
	*app.now = value
	app.service.Now = func() time.Time { return *app.now }
}

func (app *reactionApp) heartbeat(t *testing.T) {
	t.Helper()
	request[any](t, app.router, http.MethodPost, "/api/device/boxes/PB001/heartbeat", app.token,
		map[string]any{"firmware_version": "test-1", "uptime_ms": 1000, "wifi_rssi": -50}, http.StatusOK)
}

func (app *reactionApp) startedGame(t *testing.T, mode models.GameMode) (models.Session, models.Session) {
	t.Helper()
	host := request[models.Session](t, app.router, http.MethodPost, "/api/boxes/PB001/games", "",
		map[string]any{"name": "Réaction", "player_name": "Host", "mode": mode}, http.StatusCreated)
	guest := request[models.Session](t, app.router, http.MethodPost, "/api/games/"+host.Game.ID+"/join", "",
		map[string]string{"name": "Guest"}, http.StatusCreated)
	request[any](t, app.router, http.MethodPost, "/api/games/"+host.Game.ID+"/start", host.Token, nil, http.StatusOK)
	if _, err := app.service.TickReactions(context.Background()); err != nil {
		t.Fatal(err)
	}
	return host, guest
}

func (app *reactionApp) schedule(t *testing.T, gameID string, duel bool) models.ReactionChallenge {
	t.Helper()
	if duel {
		app.service.RandomIntN = func(limit int) int {
			if limit == 2 {
				return 1
			}
			return 0
		}
	} else {
		app.service.RandomIntN = func(int) int { return 0 }
	}
	mustExec(t, app.pool, `UPDATE reaction_game_states SET next_trigger_at=$2 WHERE game_id=$1`, gameID, app.now.Add(-time.Second))
	changes, err := app.service.TickReactions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].GameID != gameID {
		t.Fatalf("unexpected scheduler changes: %+v", changes)
	}
	var id string
	if err = app.pool.QueryRow(context.Background(), `SELECT id FROM reaction_challenges
		WHERE game_id=$1 AND status='awaiting_assignment'`, gameID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	state := request[models.ReactionState](t, app.router, http.MethodGet, "/api/games/"+gameID+"/reaction", app.hostToken(t, gameID), nil, http.StatusOK)
	if state.Challenge == nil || state.Challenge.ID != id {
		t.Fatal("REST reaction state did not expose scheduled challenge")
	}
	return *state.Challenge
}

func (app *reactionApp) hostToken(t *testing.T, gameID string) string {
	t.Helper()
	var hash string
	if err := app.pool.QueryRow(context.Background(), `SELECT token_hash FROM players WHERE game_id=$1 AND is_host`, gameID).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	// Tests call schedule only from flows that already know the host; this map is
	// populated by looking up the token through the helper below instead.
	var token string
	if value, ok := reactionHostTokens.Load(gameID); ok {
		token = value.(string)
	}
	if token == "" || services.TokenHash(token) != hash {
		t.Fatal("host token missing")
	}
	return token
}

var reactionHostTokens sync.Map

func rememberReactionHost(session models.Session) {
	reactionHostTokens.Store(session.Game.ID, session.Token)
}

func TestReactionMigrationRollbackKeepsCoreData(t *testing.T) {
	pool, dir := isolatedDB(t)
	migrateAndSeed(t, pool, dir)
	down, err := os.ReadFile(filepath.Join(dir, "migrations", "008_esp32_reaction.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, pool, string(down))
	var boxName string
	if err = pool.QueryRow(context.Background(), `SELECT name FROM boxes WHERE id='PB001'`).Scan(&boxName); err != nil {
		t.Fatal(err)
	}
	var tables int
	if err = pool.QueryRow(context.Background(), `SELECT count(*) FROM information_schema.tables
		WHERE table_schema=current_schema() AND table_name IN
		('box_devices','device_commands','reaction_game_states','reaction_challenges')`).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 0 || boxName == "" {
		t.Fatalf("rollback tables=%d box=%q", tables, boxName)
	}
}

func TestDeviceAuthenticationAndHeartbeat(t *testing.T) {
	app := newReactionApp(t)
	path := "/api/device/boxes/PB001/heartbeat"
	body := map[string]any{"firmware_version": "test-1", "uptime_ms": 1000, "wifi_rssi": -45}
	request[any](t, app.router, http.MethodPost, path, "", body, http.StatusUnauthorized)
	request[any](t, app.router, http.MethodPost, path, "bad", body, http.StatusUnauthorized)
	request[any](t, app.router, http.MethodPost, path, app.token, body, http.StatusOK)

	var seen time.Time
	var firmware string
	if err := app.pool.QueryRow(context.Background(), `SELECT last_seen_at,firmware_version FROM box_devices WHERE box_id='PB001'`).Scan(&seen, &firmware); err != nil {
		t.Fatal(err)
	}
	if !seen.Equal(*app.now) || firmware != "test-1" {
		t.Fatalf("heartbeat not stored: %s %s", seen, firmware)
	}
	var events int
	if err := app.pool.QueryRow(context.Background(), `SELECT count(*) FROM game_events`).Scan(&events); err != nil || events != 0 {
		t.Fatalf("heartbeat created game events: count=%d err=%v", events, err)
	}

	mustExec(t, app.pool, `INSERT INTO boxes(id,name) VALUES('PB002','Other box')`)
	otherToken := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))
	if err := app.service.ProvisionDevice(context.Background(), "PB002", otherToken); err != nil {
		t.Fatal(err)
	}
	request[any](t, app.router, http.MethodPost, "/api/device/boxes/PB002/heartbeat", app.token, body, http.StatusUnauthorized)
	mustExec(t, app.pool, `UPDATE box_devices SET enabled=false WHERE box_id='PB001'`)
	request[any](t, app.router, http.MethodPost, path, app.token, body, http.StatusUnauthorized)
}

func TestReactionSchedulerLifecycle(t *testing.T) {
	app := newReactionApp(t)
	host := request[models.Session](t, app.router, http.MethodPost, "/api/boxes/PB001/games", "",
		map[string]any{"name": "Lobby", "player_name": "Host", "mode": models.ModeSecretMissions}, http.StatusCreated)
	rememberReactionHost(host)
	if changes, err := app.service.TickReactions(context.Background()); err != nil || len(changes) != 0 {
		t.Fatalf("lobby scheduled reaction: %+v %v", changes, err)
	}
	guest := request[models.Session](t, app.router, http.MethodPost, "/api/games/"+host.Game.ID+"/join", "",
		map[string]string{"name": "Guest"}, http.StatusCreated)
	_ = guest
	request[any](t, app.router, http.MethodPost, "/api/games/"+host.Game.ID+"/start", host.Token, nil, http.StatusOK)
	if _, err := app.service.TickReactions(context.Background()); err != nil {
		t.Fatal(err)
	}
	var next time.Time
	if err := app.pool.QueryRow(context.Background(), `SELECT next_trigger_at FROM reaction_game_states WHERE game_id=$1`, host.Game.ID).Scan(&next); err != nil || !next.After(*app.now) {
		t.Fatalf("persistent schedule missing: %v %v", next, err)
	}
	mustExec(t, app.pool, `UPDATE reaction_game_states SET next_trigger_at=$2 WHERE game_id=$1`, host.Game.ID, app.now.Add(-time.Second))
	if changes, err := app.service.TickReactions(context.Background()); err != nil || len(changes) != 0 {
		t.Fatalf("offline box scheduled reaction: %+v %v", changes, err)
	}
	app.heartbeat(t)
	challenge := app.schedule(t, host.Game.ID, false)
	mustExec(t, app.pool, `UPDATE reaction_game_states SET next_trigger_at=$2 WHERE game_id=$1`, host.Game.ID, app.now.Add(-time.Second))
	if _, err := app.service.TickReactions(context.Background()); err != nil {
		t.Fatal(err)
	}
	var active int
	if err := app.pool.QueryRow(context.Background(), `SELECT count(*) FROM reaction_challenges WHERE game_id=$1 AND status IN ('awaiting_assignment','awaiting_device','armed')`, host.Game.ID).Scan(&active); err != nil || active != 1 {
		t.Fatalf("active challenges=%d err=%v", active, err)
	}
	app.setNow(challenge.ExpiresAt.Add(time.Second))
	if _, err := app.service.TickReactions(context.Background()); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := app.pool.QueryRow(context.Background(), `SELECT status FROM reaction_challenges WHERE id=$1`, challenge.ID).Scan(&status); err != nil || status != "expired" {
		t.Fatalf("expired status=%s err=%v", status, err)
	}
	app.heartbeat(t)
	second := app.schedule(t, host.Game.ID, false)
	request[any](t, app.router, http.MethodPost, "/api/games/"+host.Game.ID+"/end", host.Token, nil, http.StatusOK)
	if err := app.pool.QueryRow(context.Background(), `SELECT status FROM reaction_challenges WHERE id=$1`, second.ID).Scan(&status); err != nil || status != "cancelled" {
		t.Fatalf("ended game challenge=%s err=%v", status, err)
	}
	if changes, err := app.service.TickReactions(context.Background()); err != nil || len(changes) != 0 {
		t.Fatalf("ended game scheduled reaction: %+v %v", changes, err)
	}
}

func TestReactionAssignmentCommandsAndResults(t *testing.T) {
	app := newReactionApp(t)
	host, guest := app.startedGame(t, models.ModeChaos)
	rememberReactionHost(host)
	app.heartbeat(t)
	challenge := app.schedule(t, host.Game.ID, true)
	path := "/api/games/" + host.Game.ID + "/reaction/" + challenge.ID + "/assign"
	request[any](t, app.router, http.MethodPost, path, guest.Token,
		map[string]any{"button_s2_player_id": host.Player.ID, "button_s3_player_id": guest.Player.ID}, http.StatusForbidden)
	request[any](t, app.router, http.MethodPost, path, host.Token,
		map[string]any{"button_s2_player_id": host.Player.ID, "button_s3_player_id": host.Player.ID}, http.StatusBadRequest)
	var foreignPlayerID string
	mustExec(t, app.pool, `INSERT INTO boxes(id,name) VALUES('PB002','Other');
		INSERT INTO games(id,box_id,name,mode,status) VALUES('00000000-0000-0000-0000-000000000021','PB002','Other','chaos','lobby')`)
	if err := app.pool.QueryRow(context.Background(), `INSERT INTO players(id,game_id,name,token_hash,is_host)
		VALUES(gen_random_uuid(),'00000000-0000-0000-0000-000000000021','Foreign','foreign-hash',true) RETURNING id`).Scan(&foreignPlayerID); err != nil {
		t.Fatal(err)
	}
	request[any](t, app.router, http.MethodPost, path, host.Token,
		map[string]any{"button_s2_player_id": host.Player.ID, "button_s3_player_id": foreignPlayerID}, http.StatusForbidden)
	assigned := request[models.ReactionChallenge](t, app.router, http.MethodPost, path, host.Token,
		map[string]any{"button_s2_player_id": host.Player.ID, "button_s3_player_id": guest.Player.ID}, http.StatusOK)
	if assigned.Status != "awaiting_device" {
		t.Fatalf("assigned status=%s", assigned.Status)
	}
	request[any](t, app.router, http.MethodPost, path, host.Token,
		map[string]any{"button_s2_player_id": host.Player.ID, "button_s3_player_id": guest.Player.ID}, http.StatusConflict)

	commands := request[struct{ Commands []models.DeviceCommand }](t, app.router, http.MethodGet,
		"/api/device/boxes/PB001/commands", app.token, nil, http.StatusOK).Commands
	if len(commands) != 1 || commands[0].BoxID != "PB001" || commands[0].ChallengeID != challenge.ID {
		t.Fatalf("commands=%+v", commands)
	}
	ackPath := "/api/device/boxes/PB001/commands/" + commands[0].ID + "/ack"
	request[any](t, app.router, http.MethodPost, ackPath, app.token, nil, http.StatusOK)
	request[any](t, app.router, http.MethodPost, ackPath, app.token, nil, http.StatusOK)

	reactionMS := 243
	resultBody := map[string]any{
		"event_id": "PB001-test-1", "challenge_id": challenge.ID, "command_id": commands[0].ID,
		"button": "S3", "reaction_ms": reactionMS, "false_start": false, "timeout": false,
	}
	responses := make(chan *httptest.ResponseRecorder, 4)
	var group sync.WaitGroup
	for range 4 {
		group.Go(func() {
			responses <- call(app.router, http.MethodPost, "/api/device/boxes/PB001/reaction-results", app.token, resultBody)
		})
	}
	group.Wait()
	close(responses)
	newResults := 0
	for response := range responses {
		if response.Code != http.StatusOK {
			t.Fatalf("concurrent result: %s", response.Body.String())
		}
		var result models.ReactionResult
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if !result.AlreadyProcessed {
			newResults++
		}
	}
	if newResults != 1 {
		t.Fatalf("non-duplicate results=%d", newResults)
	}
	player := request[models.Player](t, app.router, http.MethodGet, "/api/players/me", guest.Token, nil, http.StatusOK)
	if player.Score != 100 || player.CompletedMissions != 0 {
		t.Fatalf("reaction changed wrong counters: %+v", player)
	}
	var resolvedEvents, missionEvents, chaosEvents int
	if err := app.pool.QueryRow(context.Background(), `SELECT
		count(*) FILTER (WHERE type='reaction_challenge_resolved'),
		count(*) FILTER (WHERE type='mission_completed'),
		count(*) FILTER (WHERE type IN ('chaos_event_triggered','chaos_event_consumed'))
		FROM game_events WHERE game_id=$1`, host.Game.ID).Scan(&resolvedEvents, &missionEvents, &chaosEvents); err != nil {
		t.Fatal(err)
	}
	if resolvedEvents != 1 || missionEvents != 0 || chaosEvents != 0 {
		t.Fatalf("unexpected gameplay events: reaction=%d mission=%d chaos=%d", resolvedEvents, missionEvents, chaosEvents)
	}

	// A duel false start awards the opponent exactly once.
	app.heartbeat(t)
	falseStartChallenge := app.schedule(t, host.Game.ID, true)
	falsePath := "/api/games/" + host.Game.ID + "/reaction/" + falseStartChallenge.ID + "/assign"
	request[any](t, app.router, http.MethodPost, falsePath, host.Token,
		map[string]any{"button_s2_player_id": host.Player.ID, "button_s3_player_id": guest.Player.ID}, http.StatusOK)
	commands = request[struct{ Commands []models.DeviceCommand }](t, app.router, http.MethodGet,
		"/api/device/boxes/PB001/commands", app.token, nil, http.StatusOK).Commands
	command := commands[len(commands)-1]
	request[any](t, app.router, http.MethodPost, "/api/device/boxes/PB001/commands/"+command.ID+"/ack", app.token, nil, http.StatusOK)
	falseResult := request[models.ReactionResult](t, app.router, http.MethodPost,
		"/api/device/boxes/PB001/reaction-results", app.token, map[string]any{
			"event_id": "PB001-test-2", "challenge_id": falseStartChallenge.ID, "command_id": command.ID,
			"button": "S2", "false_start": true, "timeout": false,
		}, http.StatusOK)
	if falseResult.Challenge.WinnerPlayerID == nil || *falseResult.Challenge.WinnerPlayerID != guest.Player.ID ||
		falseResult.Challenge.AwardedPoints != 100 || falseResult.Challenge.FalseStartPlayerID == nil ||
		*falseResult.Challenge.FalseStartPlayerID != host.Player.ID {
		t.Fatalf("invalid false-start result: %+v", falseResult)
	}

	// Solo commands reject early/unassigned results and accept a timeout for 0 point.
	app.heartbeat(t)
	solo := app.schedule(t, host.Game.ID, false)
	soloPath := "/api/games/" + host.Game.ID + "/reaction/" + solo.ID + "/assign"
	request[any](t, app.router, http.MethodPost, soloPath, host.Token,
		map[string]any{"button_s2_player_id": host.Player.ID, "button_s3_player_id": nil}, http.StatusOK)
	commands = request[struct{ Commands []models.DeviceCommand }](t, app.router, http.MethodGet,
		"/api/device/boxes/PB001/commands", app.token, nil, http.StatusOK).Commands
	command = commands[len(commands)-1]
	early := map[string]any{
		"event_id": "PB001-test-3", "challenge_id": solo.ID, "command_id": command.ID,
		"button": "S2", "reaction_ms": 180, "false_start": false, "timeout": false,
	}
	request[any](t, app.router, http.MethodPost, "/api/device/boxes/PB001/reaction-results", app.token, early, http.StatusConflict)
	request[any](t, app.router, http.MethodPost, "/api/device/boxes/PB001/commands/"+command.ID+"/ack", app.token, nil, http.StatusOK)
	unassigned := map[string]any{
		"event_id": "PB001-test-4", "challenge_id": solo.ID, "command_id": command.ID,
		"button": "S3", "reaction_ms": 180, "false_start": false, "timeout": false,
	}
	request[any](t, app.router, http.MethodPost, "/api/device/boxes/PB001/reaction-results", app.token, unassigned, http.StatusBadRequest)
	timeoutResult := request[models.ReactionResult](t, app.router, http.MethodPost,
		"/api/device/boxes/PB001/reaction-results", app.token, map[string]any{
			"event_id": "PB001-test-5", "challenge_id": solo.ID, "command_id": command.ID,
			"false_start": false, "timeout": true,
		}, http.StatusOK)
	if timeoutResult.Challenge.AwardedPoints != 0 || timeoutResult.Challenge.WinnerPlayerID != nil {
		t.Fatalf("solo timeout awarded points: %+v", timeoutResult)
	}
	request[any](t, app.router, http.MethodPost, "/api/device/boxes/PB001/reaction-results", app.token,
		map[string]any{"event_id": "PB001-test-x", "challenge_id": "00000000-0000-0000-0000-000000000099", "command_id": command.ID, "timeout": true}, http.StatusNotFound)

	// Polling hides and marks expired commands.
	app.heartbeat(t)
	expiring := app.schedule(t, host.Game.ID, false)
	expiringPath := "/api/games/" + host.Game.ID + "/reaction/" + expiring.ID + "/assign"
	request[any](t, app.router, http.MethodPost, expiringPath, host.Token,
		map[string]any{"button_s2_player_id": host.Player.ID, "button_s3_player_id": nil}, http.StatusOK)
	var expiringCommandID string
	if err := app.pool.QueryRow(context.Background(), `SELECT id FROM device_commands WHERE challenge_id=$1`, expiring.ID).Scan(&expiringCommandID); err != nil {
		t.Fatal(err)
	}
	mustExec(t, app.pool, `UPDATE device_commands SET expires_at=$2 WHERE id=$1`, expiringCommandID, app.now.Add(-time.Second))
	commands = request[struct{ Commands []models.DeviceCommand }](t, app.router, http.MethodGet,
		"/api/device/boxes/PB001/commands", app.token, nil, http.StatusOK).Commands
	if len(commands) != 0 {
		t.Fatalf("expired command returned by poll: %+v", commands)
	}
	var commandStatus string
	if err := app.pool.QueryRow(context.Background(), `SELECT status FROM device_commands WHERE id=$1`, expiringCommandID).Scan(&commandStatus); err != nil || commandStatus != "expired" {
		t.Fatalf("expired command status=%s err=%v", commandStatus, err)
	}
}
