package services

import (
	"context"
	"fmt"

	"partybox/backend/internal/chaos"
	"partybox/backend/internal/models"
)

func (s *Service) ChaosState(ctx context.Context, gameID string, player models.Player) (chaos.Snapshot, error) {
	game, err := s.Game(ctx, gameID, player)
	if err != nil {
		return chaos.Snapshot{}, err
	}
	if game.Mode != models.ModeChaos {
		return chaos.Snapshot{}, fmt.Errorf("%w : cette partie n’utilise pas le mode Chaos", models.ErrConflict)
	}
	return s.chaosEngine().GetState(ctx, s.Repo, gameID)
}
