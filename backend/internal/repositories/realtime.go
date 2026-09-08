package repositories

import (
	"context"

	"partybox/backend/internal/models"
)

func (r *Repository) Player(ctx context.Context, id string) (models.Player, error) {
	return player(ctx, r.Pool, id)
}
