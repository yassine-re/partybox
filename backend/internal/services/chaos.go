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
	state, err := s.chaosEngine().GetState(ctx, s.Repo, gameID)
	if err == nil && game.Status == "playing" {
		state.Progress.TriggerAt = chaosThreshold(game, s.now())
		state.Progress.Completed = min(state.Progress.Completed, state.Progress.TriggerAt)
	}
	return state, err
}
