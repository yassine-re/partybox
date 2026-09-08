package services

import (
	"context"

	"partybox/backend/internal/models"
)

func (s *Service) AuthorizeRealtime(ctx context.Context, gameID, playerID string) error {
	p, err := s.Repo.Player(ctx, playerID)
	if err != nil {
		return err
	}
	if p.GameID != gameID {
		return models.ErrForbidden
	}
	return nil
}
