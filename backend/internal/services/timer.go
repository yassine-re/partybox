package services

import (
	"context"
	"partybox/backend/internal/models"
	"partybox/backend/internal/repositories"
)

type TimedChange struct {
	GameID string
	Type   models.GameEventType
}

func (s *Service) FinishDueGame(ctx context.Context, gameID string) (bool, error) {
	changes, err := s.tickGame(ctx, gameID, false)
	if err != nil {
		return false, err
	}
	for _, change := range changes {
		if change.Type == models.GameEventGameEnded {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) TickGames(ctx context.Context) ([]TimedChange, error) {
	ids, err := s.Repo.TimedGameIDs(ctx)
	if err != nil {
		return nil, err
	}
	all := []TimedChange{}
	for _, id := range ids {
		changes, tickErr := s.tickGame(ctx, id, true)
		if tickErr != nil {
			return all, tickErr
		}
		all = append(all, changes...)
	}
	return all, nil
}

func (s *Service) tickGame(ctx context.Context, gameID string, phases bool) ([]TimedChange, error) {
	changes := []TimedChange{}
	err := s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
		g, err := tx.LockGame(ctx, gameID)
		if err != nil {
			return err
		}
		if g.Status != "playing" || g.EndsAt == nil {
			return nil
		}
		now := s.now()
		if gameExpired(g, now) {
			if err = tx.SetGameStatus(ctx, gameID, "ended", now); err != nil {
				return err
			}
			if err = tx.ExpireValidations(ctx, gameID, now); err != nil {
				return err
			}
			cancelled, cancelErr := tx.CancelActiveReactions(ctx, gameID, now)
			if cancelErr != nil {
				return cancelErr
			}
			for _, challengeID := range cancelled {
				challenge, challengeErr := tx.LockReactionChallenge(ctx, challengeID)
				if challengeErr != nil {
					return challengeErr
				}
				if challengeErr = tx.RecordReactionEvent(ctx, challenge, models.GameEventReactionChallengeExpired, nil, map[string]any{"status": "cancelled"}); challengeErr != nil {
					return challengeErr
				}
			}
			if err = tx.RecordGameEvent(ctx, &models.GameEvent{GameID: gameID, Type: models.GameEventGameEnded}); err != nil {
				return err
			}
			changes = append(changes, TimedChange{gameID, models.GameEventGameEnded})
			return nil
		}
		if !phases {
			return nil
		}
		remaining := g.EndsAt.Sub(now)
		for _, phase := range []struct {
			seconds int
			kind    models.GameEventType
		}{{300, models.GameEventFiveMinutes}, {60, models.GameEventFinalMinute}} {
			if remaining.Seconds() > float64(phase.seconds) {
				continue
			}
			found, checkErr := tx.HasGameEvent(ctx, gameID, phase.kind)
			if checkErr != nil {
				return checkErr
			}
			if found {
				continue
			}
			if err = tx.RecordGameEvent(ctx, &models.GameEvent{GameID: gameID, Type: phase.kind}); err != nil {
				return err
			}
			changes = append(changes, TimedChange{gameID, phase.kind})
		}
		return nil
	})
	return changes, err
}
