package services

import (
	"context"
	"encoding/json"
	"fmt"

	"partybox/backend/internal/models"
	"partybox/backend/internal/repositories"
)

func (s *Service) PendingValidations(ctx context.Context, gameID string, viewer models.Player) ([]models.ValidationRequest, error) {
	if viewer.GameID != gameID {
		return nil, models.ErrForbidden
	}
	if _, err := s.FinishDueGame(ctx, gameID); err != nil {
		return nil, err
	}
	g, err := s.Repo.Game(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if g.Status != "playing" {
		return []models.ValidationRequest{}, nil
	}
	return s.Repo.PendingValidations(ctx, gameID, viewer.ID)
}

func (s *Service) ResolveValidation(ctx context.Context, gameID, requestID string, validator models.Player, approved bool) (models.Completion, error) {
	if validator.GameID != gameID {
		return models.Completion{}, models.ErrForbidden
	}
	if _, err := s.FinishDueGame(ctx, gameID); err != nil {
		return models.Completion{}, err
	}
	result := models.Completion{}
	err := s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
		g, err := tx.LockGame(ctx, gameID)
		if err != nil {
			return err
		}
		v, err := tx.Validation(ctx, requestID)
		if err != nil {
			return err
		}
		if v.GameID != gameID {
			return models.ErrNotFound
		}
		if v.PlayerID == validator.ID {
			return fmt.Errorf("%w : tu ne peux pas valider ta propre mission", models.ErrForbidden)
		}
		if v.Status != "pending" {
			return models.ErrConflict
		}
		now := s.now()
		if g.Status != "playing" || gameExpired(g, now) {
			return models.ErrConflict
		}
		status, points, err := tx.Assignment(ctx, v.AssignmentID, v.PlayerID)
		if err != nil {
			return err
		}
		if status != "assigned" {
			return models.ErrConflict
		}
		resolution := "rejected"
		if approved {
			resolution = "approved"
		}
		if approved {
			if g.Mode == models.ModeTreasureHunt && s.Vision != nil {
				proof, proofErr := tx.ValidProof(ctx, v.AssignmentID)
				if proofErr != nil {
					return proofErr
				}
				if proof == nil {
					return models.ErrConflict
				}
			}
			awarded := points
			allowChaos := true
			if g.Mode == models.ModeChaos {
				effect, effectErr := s.chaosEngine().ApplyCompletion(ctx, tx, g.ID, v.PlayerID, points)
				if effectErr != nil {
					return effectErr
				}
				awarded = effect.AwardedPoints
				allowChaos = !effect.HadBlockingEvent
			}
			if gameFinalMinute(g, now) {
				awarded *= 2
			}
			if err = tx.CompleteAssignment(ctx, v.AssignmentID, awarded, now); err != nil {
				return err
			}
			if err = tx.AddScore(ctx, v.PlayerID, awarded); err != nil {
				return err
			}
			if err = tx.AssignMission(ctx, v.PlayerID); err != nil {
				return err
			}
			payload, _ := json.Marshal(map[string]any{"assignment_id": v.AssignmentID, "points": awarded})
			playerID := v.PlayerID
			if err = tx.RecordGameEvent(ctx, &models.GameEvent{GameID: gameID, PlayerID: &playerID, Type: models.GameEventMissionCompleted, Payload: payload}); err != nil {
				return err
			}
			if g.Mode == models.ModeChaos {
				if err = s.chaosEngine().MaybeTriggerEvent(ctx, tx, g.ID, allowChaos, chaosThreshold(g, now)); err != nil {
					return err
				}
			}
			result.AwardedPoints = awarded
		}
		if err = tx.ResolveValidation(ctx, requestID, resolution, validator.ID, now); err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]any{"request_id": requestID, "approved": approved})
		if err = tx.RecordGameEvent(ctx, &models.GameEvent{GameID: gameID, PlayerID: &validator.ID, Type: models.GameEventValidationResolved, Payload: payload}); err != nil {
			return err
		}
		result.Player, err = tx.Player(ctx, v.PlayerID)
		if err != nil {
			return err
		}
		if g.LeaderboardVisibility == "hidden" {
			result.Player.Score = 0
			result.Player.CompletedMissions = 0
		}
		return nil
	})
	return result, err
}

func (s *Service) FinalRecap(ctx context.Context, gameID string, viewer models.Player) (models.FinalRecap, error) {
	if viewer.GameID != gameID {
		return models.FinalRecap{}, models.ErrForbidden
	}
	if _, err := s.FinishDueGame(ctx, gameID); err != nil {
		return models.FinalRecap{}, err
	}
	status, err := s.Repo.GameStatus(ctx, gameID)
	if err != nil {
		return models.FinalRecap{}, err
	}
	if status != "ended" {
		return models.FinalRecap{}, models.ErrConflict
	}
	return s.Repo.FinalRecap(ctx, gameID)
}
