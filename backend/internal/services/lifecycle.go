package services

import (
	"context"
	"encoding/json"
	"fmt"
	"partybox/backend/internal/models"
	"partybox/backend/internal/repositories"
)

func (s *Service) create(ctx context.Context, boxID, name, hostName, hash string, mode models.GameMode) (models.Session, error) {
	var result models.Session
	err := s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
		if err := tx.LockBox(ctx, boxID); err != nil {
			return err
		}
		var err error
		result.Game, err = tx.InsertGame(ctx, boxID, name, mode)
		if err != nil {
			return err
		}
		result.Player, err = tx.InsertPlayer(ctx, result.Game.ID, hostName, hash, true)
		return err
	})
	return result, err
}

func (s *Service) join(ctx context.Context, gameID, name, hash string) (models.Session, error) {
	var result models.Session
	err := s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
		g, err := tx.LockGame(ctx, gameID)
		if err != nil {
			return err
		}
		if g.Status == "ended" {
			return fmt.Errorf("%w : cette partie est terminée", models.ErrConflict)
		}
		result.Game = g
		result.Player, err = tx.InsertPlayer(ctx, gameID, name, hash, false)
		if err != nil {
			return err
		}
		if g.Status == "playing" {
			if err = tx.AssignMission(ctx, result.Player.ID); err != nil {
				return err
			}
		}
		playerID := result.Player.ID
		return tx.RecordGameEvent(ctx, &models.GameEvent{
			GameID: gameID, PlayerID: &playerID, Type: models.GameEventPlayerJoined,
		})
	})
	return result, err
}

func (s *Service) transition(ctx context.Context, gameID string, p models.Player, target string) error {
	if p.GameID != gameID || !p.IsHost {
		return models.ErrForbidden
	}
	return s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
		g, err := tx.LockGame(ctx, gameID)
		if err != nil {
			return err
		}
		if g.Status == target {
			return nil
		} // Host actions are safe to retry.
		if target == "playing" {
			if g.Status != "lobby" {
				return models.ErrConflict
			}
			list, err := tx.Players(ctx, gameID)
			if err != nil {
				return err
			}
			definition, supported := models.LookupGameMode(g.Mode)
			if !supported {
				return fmt.Errorf("%w : mode de jeu non supporté", models.ErrConflict)
			}
			if len(list) < definition.MinPlayers {
				return fmt.Errorf("%w : il faut au moins %d joueurs", models.ErrConflict, definition.MinPlayers)
			}
			for _, member := range list {
				if err = tx.AssignMission(ctx, member.ID); err != nil {
					return err
				}
			}
		} else if g.Status != "playing" && g.Status != "lobby" {
			return models.ErrConflict
		}
		if err = tx.SetGameStatus(ctx, gameID, target); err != nil {
			return err
		}
		eventType := models.GameEventGameEnded
		if target == "playing" {
			eventType = models.GameEventGameStarted
		}
		playerID := p.ID
		return tx.RecordGameEvent(ctx, &models.GameEvent{
			GameID: gameID, PlayerID: &playerID, Type: eventType,
		})
	})
}

func (s *Service) complete(ctx context.Context, p models.Player, assignmentID string) (models.Completion, error) {
	var result models.Completion
	err := s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
		// All mutations take this lock first, including End: no score can be
		// added after the final ranking has been frozen.
		g, err := tx.LockGame(ctx, p.GameID)
		if err != nil {
			return err
		}
		status, points, err := tx.Assignment(ctx, assignmentID, p.ID)
		if err != nil {
			return err
		}
		if status == "completed" {
			result.AlreadyCompleted = true
		} else {
			if g.Status != "playing" {
				return models.ErrConflict
			}
			if err = tx.CompleteAssignment(ctx, assignmentID); err != nil {
				return err
			}
			if err = tx.AddScore(ctx, p.ID, points); err != nil {
				return err
			}
			if err = tx.AssignMission(ctx, p.ID); err != nil {
				return err
			}
			payload, marshalErr := json.Marshal(struct {
				AssignmentID string `json:"assignment_id"`
				Points       int    `json:"points"`
			}{AssignmentID: assignmentID, Points: points})
			if marshalErr != nil {
				return marshalErr
			}
			playerID := p.ID
			if err = tx.RecordGameEvent(ctx, &models.GameEvent{
				GameID: p.GameID, PlayerID: &playerID,
				Type: models.GameEventMissionCompleted, Payload: payload,
			}); err != nil {
				return err
			}
			result.AwardedPoints = points
		}
		result.Player, err = tx.Player(ctx, p.ID)
		if err != nil {
			return err
		}
		if g.Status == "playing" {
			result.Mission, err = tx.CurrentMission(ctx, p.ID)
		}
		return err
	})
	return result, err
}
