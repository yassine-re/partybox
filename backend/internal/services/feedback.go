package services

import (
	"context"
	"encoding/json"
	"fmt"

	"partybox/backend/internal/models"
	"partybox/backend/internal/repositories"
)

func (s *Service) SetMissionFeedback(ctx context.Context, p models.Player, assignmentID string, rating int) error {
	if len(assignmentID) != 36 || rating < -1 || rating > 1 {
		return models.ErrInvalid
	}

	return s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
		status, _, err := tx.Assignment(ctx, assignmentID, p.ID)
		if err != nil {
			return err
		}
		if status != "completed" {
			return fmt.Errorf("%w : seule une mission terminée peut être notée", models.ErrConflict)
		}

		changed, err := tx.SetMissionFeedback(ctx, assignmentID, rating)
		if err != nil || !changed {
			return err
		}
		payload, err := json.Marshal(struct {
			AssignmentID string `json:"assignment_id"`
			Rating       int    `json:"rating"`
		}{AssignmentID: assignmentID, Rating: rating})
		if err != nil {
			return err
		}
		playerID := p.ID
		return tx.RecordGameEvent(ctx, &models.GameEvent{
			GameID: p.GameID, PlayerID: &playerID,
			Type: models.GameEventMissionFeedbackSubmitted, Payload: payload,
		})
	})
}
