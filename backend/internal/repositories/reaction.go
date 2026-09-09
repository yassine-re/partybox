package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"partybox/backend/internal/models"
)

const reactionChallengeColumns = `rc.id,rc.game_id,rc.box_id,rc.kind,rc.status,
	rc.button_s2_player_id,s2.name,rc.button_s3_player_id,s3.name,rc.delay_ms,
	rc.winner_player_id,winner.name,rc.false_start_player_id,false_starter.name,
	rc.reaction_ms,rc.awarded_points,rc.result_event_id,rc.scheduled_at,rc.assigned_at,rc.armed_at,rc.resolved_at,rc.expires_at`

type rowScanner interface {
	Scan(...any) error
}

func scanReaction(row rowScanner) (models.ReactionChallenge, error) {
	var challenge models.ReactionChallenge
	err := row.Scan(
		&challenge.ID, &challenge.GameID, &challenge.BoxID, &challenge.Kind, &challenge.Status,
		&challenge.ButtonS2PlayerID, &challenge.ButtonS2PlayerName,
		&challenge.ButtonS3PlayerID, &challenge.ButtonS3PlayerName, &challenge.DelayMS,
		&challenge.WinnerPlayerID, &challenge.WinnerPlayerName,
		&challenge.FalseStartPlayerID, &challenge.FalseStartPlayerName,
		&challenge.ReactionMS, &challenge.AwardedPoints, &challenge.ResultEventID, &challenge.ScheduledAt,
		&challenge.AssignedAt, &challenge.ArmedAt, &challenge.ResolvedAt, &challenge.ExpiresAt,
	)
	return challenge, normalize(err)
}

func challengeQuery(where string) string {
	return `SELECT ` + reactionChallengeColumns + ` FROM reaction_challenges rc
		LEFT JOIN players s2 ON s2.id=rc.button_s2_player_id
		LEFT JOIN players s3 ON s3.id=rc.button_s3_player_id
		LEFT JOIN players winner ON winner.id=rc.winner_player_id
		LEFT JOIN players false_starter ON false_starter.id=rc.false_start_player_id ` + where
}

func (r *Repository) ReactionState(ctx context.Context, gameID string, onlineCutoff time.Time, enabled bool) (models.ReactionState, error) {
	state := models.ReactionState{Enabled: enabled}
	var gameBoxID string
	err := r.Pool.QueryRow(ctx, `SELECT box_id FROM games WHERE id=$1`, gameID).Scan(&gameBoxID)
	if err != nil {
		return state, normalize(err)
	}
	err = r.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM box_devices
		WHERE box_id=$1 AND enabled AND last_seen_at>$2)`, gameBoxID, onlineCutoff).Scan(&state.DeviceOnline)
	if err != nil {
		return state, err
	}
	challenge, err := scanReaction(r.Pool.QueryRow(ctx, challengeQuery(
		`WHERE rc.game_id=$1 ORDER BY
		 (rc.status IN ('awaiting_assignment','awaiting_device','armed')) DESC,
		 rc.scheduled_at DESC,rc.id DESC LIMIT 1`), gameID))
	if errors.Is(err, models.ErrNotFound) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	state.Challenge = &challenge
	return state, nil
}

func (r *Repository) GamesMissingReactionSchedule(ctx context.Context) ([]string, error) {
	rows, err := r.Pool.Query(ctx, `SELECT g.id FROM games g
		LEFT JOIN reaction_game_states rs ON rs.game_id=g.id
		WHERE g.status='playing' AND rs.game_id IS NULL ORDER BY g.created_at,g.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Repository) EnsureReactionSchedule(ctx context.Context, gameID string, next time.Time) error {
	_, err := r.Pool.Exec(ctx, `INSERT INTO reaction_game_states(game_id,next_trigger_at)
		SELECT id,$2 FROM games WHERE id=$1 AND status='playing' ON CONFLICT(game_id) DO NOTHING`, gameID, next)
	return normalize(err)
}

func (r *Repository) DueReactionGames(ctx context.Context, now, onlineCutoff time.Time) ([]string, error) {
	rows, err := r.Pool.Query(ctx, `SELECT rs.game_id FROM reaction_game_states rs
		JOIN games g ON g.id=rs.game_id
		JOIN box_devices d ON d.box_id=g.box_id
		WHERE g.status='playing' AND rs.next_trigger_at IS NOT NULL AND rs.next_trigger_at<=$1
		  AND d.enabled AND d.last_seen_at>$2
		  AND NOT EXISTS (SELECT 1 FROM reaction_challenges rc WHERE rc.game_id=g.id
		    AND rc.status IN ('awaiting_assignment','awaiting_device','armed'))
		ORDER BY rs.next_trigger_at,rs.game_id LIMIT 50`, now, onlineCutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Repository) ExpirableReactionChallenges(ctx context.Context, now time.Time) ([]string, error) {
	rows, err := r.Pool.Query(ctx, `SELECT rc.id FROM reaction_challenges rc JOIN games g ON g.id=rc.game_id
		WHERE rc.status IN ('awaiting_assignment','awaiting_device','armed')
		AND (rc.expires_at<=$1 OR g.status<>'playing') ORDER BY rc.expires_at,rc.id LIMIT 100`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Repository) ReactionChallengeIdentity(ctx context.Context, challengeID string) (string, string, error) {
	var gameID, boxID string
	err := r.Pool.QueryRow(ctx, `SELECT game_id,box_id FROM reaction_challenges WHERE id=$1`, challengeID).
		Scan(&gameID, &boxID)
	return gameID, boxID, normalize(err)
}

func (t *Transaction) LockReactionState(ctx context.Context, gameID string) (*time.Time, error) {
	var next *time.Time
	err := t.tx.QueryRow(ctx, `SELECT next_trigger_at FROM reaction_game_states WHERE game_id=$1 FOR UPDATE`, gameID).Scan(&next)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, models.ErrNotFound
	}
	return next, err
}

func (t *Transaction) DeviceOnline(ctx context.Context, boxID string, cutoff time.Time) (bool, error) {
	var online bool
	err := t.tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM box_devices
		WHERE box_id=$1 AND enabled AND last_seen_at>$2)`, boxID, cutoff).Scan(&online)
	return online, err
}

func (t *Transaction) ActiveReactionExists(ctx context.Context, gameID string) (bool, error) {
	var exists bool
	err := t.tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM reaction_challenges WHERE game_id=$1
		AND status IN ('awaiting_assignment','awaiting_device','armed'))`, gameID).Scan(&exists)
	return exists, err
}

func (t *Transaction) InsertReactionChallenge(ctx context.Context, gameID, boxID, kind string, delayMS int, now, expiresAt time.Time) (models.ReactionChallenge, error) {
	var id string
	err := t.tx.QueryRow(ctx, `INSERT INTO reaction_challenges(game_id,box_id,kind,status,delay_ms,scheduled_at,expires_at)
		VALUES($1,$2,$3,'awaiting_assignment',$4,$5,$6) RETURNING id`, gameID, boxID, kind, delayMS, now, expiresAt).Scan(&id)
	if err != nil {
		return models.ReactionChallenge{}, normalize(err)
	}
	if _, err = t.tx.Exec(ctx, `UPDATE reaction_game_states SET next_trigger_at=NULL,updated_at=$2 WHERE game_id=$1`, gameID, now); err != nil {
		return models.ReactionChallenge{}, err
	}
	return scanReaction(t.tx.QueryRow(ctx, challengeQuery(`WHERE rc.id=$1`), id))
}

func (t *Transaction) LockReactionChallenge(ctx context.Context, challengeID string) (models.ReactionChallenge, error) {
	return scanReaction(t.tx.QueryRow(ctx, challengeQuery(`WHERE rc.id=$1 FOR UPDATE OF rc`), challengeID))
}

func (t *Transaction) LockReactionChallengeByCommand(ctx context.Context, commandID string) (models.ReactionChallenge, error) {
	return scanReaction(t.tx.QueryRow(ctx, challengeQuery(
		`JOIN device_commands dc ON dc.challenge_id=rc.id WHERE dc.id=$1 FOR UPDATE OF rc`), commandID))
}

func (t *Transaction) ReactionChallengeByEvent(ctx context.Context, eventID string) (models.ReactionChallenge, error) {
	return scanReaction(t.tx.QueryRow(ctx, challengeQuery(`WHERE rc.result_event_id=$1`), eventID))
}

func (t *Transaction) PlayerBelongsToGame(ctx context.Context, playerID, gameID string) (bool, error) {
	var exists bool
	err := t.tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM players WHERE id=$1 AND game_id=$2)`, playerID, gameID).Scan(&exists)
	return exists, err
}

func (t *Transaction) AssignReaction(ctx context.Context, challenge models.ReactionChallenge, s2, s3 *string, now, expiresAt time.Time) (models.DeviceCommand, error) {
	var command models.DeviceCommand
	payload, err := json.Marshal(map[string]any{
		"challenge_id":      challenge.ID,
		"kind":              challenge.Kind,
		"delay_ms":          challenge.DelayMS,
		"button_s2_enabled": s2 != nil,
		"button_s3_enabled": s3 != nil,
	})
	if err != nil {
		return command, err
	}
	err = t.tx.QueryRow(ctx, `UPDATE reaction_challenges SET status='awaiting_device',
		button_s2_player_id=$2,button_s3_player_id=$3,assigned_at=$4,expires_at=$5
		WHERE id=$1 RETURNING id`, challenge.ID, s2, s3, now, expiresAt).Scan(&command.ChallengeID)
	if err != nil {
		return command, err
	}
	err = t.tx.QueryRow(ctx, `INSERT INTO device_commands(box_id,challenge_id,type,payload,expires_at)
		VALUES($1,$2,'reaction_arm',$3,$4)
		RETURNING id,box_id,type,status,created_at,expires_at`, challenge.BoxID, challenge.ID, payload, expiresAt).
		Scan(&command.ID, &command.BoxID, &command.Type, &command.Status, &command.CreatedAt, &command.ExpiresAt)
	if err != nil {
		return command, err
	}
	if err = json.Unmarshal(payload, &command.Payload); err != nil {
		return command, err
	}
	return command, nil
}

func (t *Transaction) CommandMatchesChallenge(ctx context.Context, boxID, commandID, challengeID string) (bool, error) {
	var exists bool
	err := t.tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM device_commands
		WHERE id=$1 AND box_id=$2 AND challenge_id=$3 AND status='acknowledged')`, commandID, boxID, challengeID).Scan(&exists)
	return exists, err
}

func (t *Transaction) ResolveReaction(ctx context.Context, challengeID, eventID string, winnerID, falseStartID *string, reactionMS *int, points int, now time.Time) error {
	if winnerID != nil && points > 0 {
		if _, err := t.tx.Exec(ctx, `UPDATE players SET score=score+$1 WHERE id=$2`, points, *winnerID); err != nil {
			return err
		}
	}
	if _, err := t.tx.Exec(ctx, `UPDATE reaction_challenges SET status='resolved',winner_player_id=$2,
		false_start_player_id=$3,reaction_ms=$4,awarded_points=$5,result_event_id=$6,resolved_at=$7,expires_at=$7
		WHERE id=$1`, challengeID, winnerID, falseStartID, reactionMS, points, eventID, now); err != nil {
		return err
	}
	_, err := t.tx.Exec(ctx, `UPDATE device_commands SET status='completed',completed_at=$2 WHERE challenge_id=$1`, challengeID, now)
	return err
}

func (t *Transaction) ExpireReaction(ctx context.Context, challengeID string, now, next time.Time) (string, string, bool, error) {
	var gameID string
	if err := t.tx.QueryRow(ctx, `SELECT game_id FROM reaction_challenges WHERE id=$1`, challengeID).Scan(&gameID); err != nil {
		return "", "", false, normalize(err)
	}
	game, err := t.LockGame(ctx, gameID)
	if err != nil {
		return "", "", false, err
	}
	challenge, err := t.LockReactionChallenge(ctx, challengeID)
	if err != nil {
		return "", "", false, err
	}
	if challenge.Status == "resolved" || challenge.Status == "expired" || challenge.Status == "cancelled" {
		return challenge.GameID, challenge.Status, false, nil
	}
	status := "expired"
	if game.Status != "playing" {
		status = "cancelled"
	}
	if _, err = t.tx.Exec(ctx, `UPDATE reaction_challenges SET status=$2,resolved_at=$3,expires_at=$3 WHERE id=$1`, challengeID, status, now); err != nil {
		return "", "", false, err
	}
	if _, err = t.tx.Exec(ctx, `UPDATE device_commands SET status='expired' WHERE challenge_id=$1 AND status IN ('pending','acknowledged')`, challengeID); err != nil {
		return "", "", false, err
	}
	if game.Status == "playing" {
		_, err = t.tx.Exec(ctx, `INSERT INTO reaction_game_states(game_id,next_trigger_at,updated_at)
			VALUES($1,$2,$3) ON CONFLICT(game_id) DO UPDATE SET next_trigger_at=excluded.next_trigger_at,updated_at=excluded.updated_at`, game.ID, next, now)
	} else {
		_, err = t.tx.Exec(ctx, `UPDATE reaction_game_states SET next_trigger_at=NULL,updated_at=$2 WHERE game_id=$1`, game.ID, now)
	}
	return game.ID, status, true, err
}

func (t *Transaction) ReplanReaction(ctx context.Context, gameID string, now, next time.Time) error {
	_, err := t.tx.Exec(ctx, `INSERT INTO reaction_game_states(game_id,next_trigger_at,updated_at)
		VALUES($1,$2,$3) ON CONFLICT(game_id) DO UPDATE SET next_trigger_at=excluded.next_trigger_at,updated_at=excluded.updated_at`, gameID, next, now)
	return err
}

func (t *Transaction) CancelActiveReactions(ctx context.Context, gameID string, now time.Time) ([]string, error) {
	rows, err := t.tx.Query(ctx, `UPDATE reaction_challenges SET status='cancelled',resolved_at=$2,expires_at=$2
		WHERE game_id=$1 AND status IN ('awaiting_assignment','awaiting_device','armed') RETURNING id`, gameID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) > 0 {
		if _, err = t.tx.Exec(ctx, `UPDATE device_commands SET status='expired'
			WHERE challenge_id=ANY($1::uuid[]) AND status IN ('pending','acknowledged')`, ids); err != nil {
			return nil, err
		}
	}
	_, err = t.tx.Exec(ctx, `UPDATE reaction_game_states SET next_trigger_at=NULL,updated_at=$2 WHERE game_id=$1`, gameID, now)
	return ids, err
}

func (t *Transaction) RecordReactionEvent(ctx context.Context, challenge models.ReactionChallenge, eventType models.GameEventType, playerID *string, extra map[string]any) error {
	payload := map[string]any{"challenge_id": challenge.ID, "kind": challenge.Kind}
	for key, value := range extra {
		payload[key] = value
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("reaction payload: %w", err)
	}
	return t.RecordGameEvent(ctx, &models.GameEvent{GameID: challenge.GameID, PlayerID: playerID, Type: eventType, Payload: raw})
}
