package services

import (
	"context"
	"encoding/json"
	"fmt"

	"partybox/backend/internal/models"
	"partybox/backend/internal/repositories"
)

// SkipTreasureMission replaces an impossible hunt without awarding points.
// The game lock serializes this with completion, peer validation and game end.
func (s *Service) SkipTreasureMission(ctx context.Context, player models.Player, assignmentID string) (models.MissionSkip, error) {
	if len(assignmentID) != 36 {
		return models.MissionSkip{}, models.ErrInvalid
	}
	if _, err := s.FinishDueGame(ctx, player.GameID); err != nil {
		return models.MissionSkip{}, err
	}
	var result models.MissionSkip
	err := s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
		game, err := tx.LockGame(ctx, player.GameID)
		if err != nil {
			return err
		}
		if game.Mode != models.ModeTreasureHunt {
			return fmt.Errorf("%w : seules les recherches Treasure Hunt peuvent être passées", models.ErrConflict)
		}
		if game.Status != "playing" || gameExpired(game, s.now()) {
			return models.ErrConflict
		}
		status, _, err := tx.Assignment(ctx, assignmentID, player.ID)
		if err != nil {
			return err
		}
		if status != "assigned" {
			return models.ErrConflict
		}
		if err = tx.CancelAssignment(ctx, assignmentID, player.ID); err != nil {
			return err
		}
		if err = tx.ExpireAssignmentValidation(ctx, assignmentID, s.now()); err != nil {
			return err
		}
		if err = tx.AssignMission(ctx, player.ID); err != nil {
			return err
		}
		mission, err := tx.CurrentMission(ctx, player.ID)
		if err != nil {
			return err
		}
		if mission == nil {
			return models.ErrConflict
		}
		result.Mission = *mission
		result.Player, err = tx.Player(ctx, player.ID)
		if err != nil {
			return err
		}
		payload, err := json.Marshal(map[string]string{"assignment_id": assignmentID, "reason": "skipped"})
		if err != nil {
			return err
		}
		return tx.RecordGameEvent(ctx, &models.GameEvent{
			GameID: player.GameID, PlayerID: &player.ID, Type: models.GameEventMissionCancelled, Payload: payload,
		})
	})
	return result, err
}
