package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"partybox/backend/internal/models"
)

func (r *Repository) TimedGameIDs(ctx context.Context) ([]string, error) {
	rows, err := r.Pool.Query(ctx, `SELECT id FROM games WHERE status='playing' AND ends_at IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (t *Transaction) ExpireValidations(ctx context.Context, gameID string, now time.Time) error {
	_, err := t.tx.Exec(ctx, `UPDATE mission_validations SET status='expired',resolved_at=$2 WHERE game_id=$1 AND status='pending'`, gameID, now)
	return err
}

func (t *Transaction) CancelAssignment(ctx context.Context, assignmentID, playerID string) error {
	command, err := t.tx.Exec(ctx, `UPDATE player_missions SET status='cancelled' WHERE id=$1 AND player_id=$2 AND status='assigned'`, assignmentID, playerID)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return models.ErrConflict
	}
	return nil
}

func (t *Transaction) ExpireAssignmentValidation(ctx context.Context, assignmentID string, now time.Time) error {
	_, err := t.tx.Exec(ctx, `UPDATE mission_validations SET status='expired',resolved_at=$2 WHERE assignment_id=$1 AND status='pending'`, assignmentID, now)
	return err
}

func (t *Transaction) HasGameEvent(ctx context.Context, gameID string, kind models.GameEventType) (bool, error) {
	var found bool
	err := t.tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM game_events WHERE game_id=$1 AND type=$2)`, gameID, kind).Scan(&found)
	return found, err
}

func (t *Transaction) RequestValidation(ctx context.Context, gameID, playerID, assignmentID string) error {
	_, err := t.tx.Exec(ctx, `INSERT INTO mission_validations(assignment_id,game_id,player_id) VALUES($1,$2,$3)
		ON CONFLICT (assignment_id) DO UPDATE SET status='pending',validator_id=NULL,resolved_at=NULL,requested_at=now()
		WHERE mission_validations.status='rejected'`, assignmentID, gameID, playerID)
	return err
}

type StoredValidation struct{ ID, AssignmentID, GameID, PlayerID, Status string }

func (t *Transaction) Validation(ctx context.Context, id string) (StoredValidation, error) {
	var v StoredValidation
	err := t.tx.QueryRow(ctx, `SELECT id,assignment_id,game_id,player_id,status FROM mission_validations WHERE id=$1`, id).Scan(&v.ID, &v.AssignmentID, &v.GameID, &v.PlayerID, &v.Status)
	return v, normalize(err)
}

func (t *Transaction) ResolveValidation(ctx context.Context, id, status, validatorID string, now time.Time) error {
	_, err := t.tx.Exec(ctx, `UPDATE mission_validations SET status=$2,validator_id=$3,resolved_at=$4 WHERE id=$1 AND status='pending'`, id, status, validatorID, now)
	return err
}

func (r *Repository) PendingValidations(ctx context.Context, gameID, viewerID string) ([]models.ValidationRequest, error) {
	rows, err := r.Pool.Query(ctx, `SELECT v.id,v.assignment_id,v.player_id,p.name,m.text,v.requested_at
		FROM mission_validations v JOIN players p ON p.id=v.player_id
		JOIN player_missions pm ON pm.id=v.assignment_id JOIN missions m ON m.id=pm.mission_id
		WHERE v.game_id=$1 AND v.player_id<>$2 AND v.status='pending' AND pm.status='assigned'
		ORDER BY v.requested_at`, gameID, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.ValidationRequest{}
	for rows.Next() {
		var v models.ValidationRequest
		if err = rows.Scan(&v.ID, &v.AssignmentID, &v.PlayerID, &v.PlayerName, &v.MissionText, &v.RequestedAt); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (r *Repository) FinalRecap(ctx context.Context, gameID string) (models.FinalRecap, error) {
	result := models.FinalRecap{Players: []models.FinalPlayer{}, ChaosEvents: []models.GameEvent{}}
	list, err := players(ctx, r.Pool, gameID)
	if err != nil {
		return result, err
	}
	for _, p := range list {
		fp := models.FinalPlayer{Player: p, Missions: []models.FinalMission{}}
		rows, queryErr := r.Pool.Query(ctx, `SELECT m.text,pm.status,m.points,pm.awarded_points,pm.completed_at FROM player_missions pm JOIN missions m ON m.id=pm.mission_id WHERE pm.player_id=$1 ORDER BY pm.assigned_at,pm.id`, p.ID)
		if queryErr != nil {
			return result, queryErr
		}
		for rows.Next() {
			var mission models.FinalMission
			if queryErr = rows.Scan(&mission.Text, &mission.Status, &mission.Points, &mission.AwardedPoints, &mission.CompletedAt); queryErr != nil {
				rows.Close()
				return result, queryErr
			}
			fp.Missions = append(fp.Missions, mission)
		}
		queryErr = rows.Err()
		rows.Close()
		if queryErr != nil {
			return result, queryErr
		}
		result.Players = append(result.Players, fp)
	}
	rows, err := r.Pool.Query(ctx, `SELECT id,game_id,player_id,type,payload,created_at FROM game_events WHERE game_id=$1 AND type=$2 ORDER BY created_at`, gameID, models.GameEventChaosTriggered)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var event models.GameEvent
		if err = rows.Scan(&event.ID, &event.GameID, &event.PlayerID, &event.Type, &event.Payload, &event.CreatedAt); err != nil {
			return result, err
		}
		result.ChaosEvents = append(result.ChaosEvents, event)
	}
	return result, rows.Err()
}

func (r *Repository) GameStatus(ctx context.Context, gameID string) (string, error) {
	var status string
	err := r.Pool.QueryRow(ctx, `SELECT status FROM games WHERE id=$1`, gameID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", models.ErrNotFound
	}
	return status, err
}
