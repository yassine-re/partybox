package repositories

import (
	"context"
	"encoding/json"
	"fmt"

	"partybox/backend/internal/models"
)

func recordGameEvent(ctx context.Context, q querier, event *models.GameEvent) error {
	if len(event.Payload) == 0 {
		event.Payload = json.RawMessage(`{}`)
	}
	if !json.Valid(event.Payload) {
		return fmt.Errorf("%w : payload d’événement invalide", models.ErrInvalid)
	}
	return q.QueryRow(ctx, `INSERT INTO game_events(id,game_id,player_id,type,payload)
		VALUES(gen_random_uuid(),$1,$2,$3,$4) RETURNING id,created_at`,
		event.GameID, event.PlayerID, event.Type, event.Payload,
	).Scan(&event.ID, &event.CreatedAt)
}

func (r *Repository) RecordGameEvent(ctx context.Context, event *models.GameEvent) error {
	return normalize(recordGameEvent(ctx, r.Pool, event))
}

func (t *Transaction) RecordGameEvent(ctx context.Context, event *models.GameEvent) error {
	return recordGameEvent(ctx, t.tx, event)
}
