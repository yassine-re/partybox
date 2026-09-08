package repositories

import (
	"context"

	"partybox/backend/internal/chaos"
)

func chaosState(ctx context.Context, q querier, gameID string, lock bool) (chaos.State, error) {
	var state chaos.State
	var eventType *string
	query := `SELECT cs.game_id,cs.event_type,cs.target_player_id::text,p.name,
		cs.remaining_uses,cs.completions_since_event,cs.payload,
		cs.event_sequence,cs.started_at,cs.updated_at
		FROM chaos_states cs LEFT JOIN players p ON p.id=cs.target_player_id
		WHERE cs.game_id=$1`
	if lock {
		query += ` FOR UPDATE OF cs`
	}
	err := q.QueryRow(ctx, query, gameID).Scan(
		&state.GameID, &eventType, &state.TargetPlayerID, &state.TargetPlayerName,
		&state.RemainingUses, &state.CompletionsSinceEvent, &state.Payload,
		&state.EventSequence, &state.StartedAt, &state.UpdatedAt,
	)
	if eventType != nil {
		value := chaos.EventType(*eventType)
		state.EventType = &value
	}
	return state, normalize(err)
}

func (r *Repository) ChaosState(ctx context.Context, gameID string) (chaos.State, error) {
	return chaosState(ctx, r.Pool, gameID, false)
}

func (t *Transaction) InitializeChaosState(ctx context.Context, gameID string) error {
	_, err := t.tx.Exec(ctx, `INSERT INTO chaos_states(game_id) VALUES($1)`, gameID)
	return err
}

func (t *Transaction) LockChaosState(ctx context.Context, gameID string) (chaos.State, error) {
	return chaosState(ctx, t.tx, gameID, true)
}

func (t *Transaction) UpdateChaosState(ctx context.Context, state chaos.State) error {
	var eventType *string
	if state.EventType != nil {
		value := string(*state.EventType)
		eventType = &value
	}
	return t.tx.QueryRow(ctx, `UPDATE chaos_states SET
		event_type=$2,target_player_id=$3,remaining_uses=$4,
		completions_since_event=$5,payload=$6,event_sequence=$7,
		started_at=$8,updated_at=now() WHERE game_id=$1
		RETURNING updated_at`,
		state.GameID, eventType, state.TargetPlayerID, state.RemainingUses,
		state.CompletionsSinceEvent, state.Payload, state.EventSequence, state.StartedAt,
	).Scan(&state.UpdatedAt)
}

func (t *Transaction) CancelCurrentMissions(ctx context.Context, gameID string) ([]chaos.CancelledMission, error) {
	rows, err := t.tx.Query(ctx, `UPDATE player_missions pm SET status='cancelled',completed_at=NULL
		FROM players p WHERE p.id=pm.player_id AND p.game_id=$1 AND pm.status='assigned'
		RETURNING pm.id::text,pm.player_id::text`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []chaos.CancelledMission{}
	for rows.Next() {
		var assignment chaos.CancelledMission
		if err = rows.Scan(&assignment.AssignmentID, &assignment.PlayerID); err != nil {
			return nil, err
		}
		result = append(result, assignment)
	}
	return result, rows.Err()
}
