package services

import (
	"context"
	"encoding/json"
	"fmt"
	"partybox/backend/internal/models"
	"partybox/backend/internal/repositories"
	"time"
)

func (s *Service) create(ctx context.Context, boxID, name, hostName, hash string, mode models.GameMode, options models.GameOptions) (models.Session, error) {
	var result models.Session
	err := s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
		if err := tx.LockBox(ctx, boxID); err != nil {
			return err
		}
		var err error
		result.Game, err = tx.InsertGame(ctx, boxID, name, mode, options)
		if err != nil {
			return err
		}
		result.Player, err = tx.InsertPlayer(ctx, result.Game.ID, hostName, hash, true)
		if err != nil {
			return err
		}
		if mode == models.ModeChaos {
			return s.chaosEngine().Initialize(ctx, tx, result.Game.ID)
		}
		return nil
	})
	return result, err
}

func (s *Service) join(ctx context.Context, gameID, name, hash string) (models.Session, error) {
	if _, err := s.FinishDueGame(ctx, gameID); err != nil {
		return models.Session{}, err
	}
	var result models.Session
	err := s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
		g, err := tx.LockGame(ctx, gameID)
		if err != nil {
			return err
		}
		if g.Status == "ended" || gameExpired(g, s.now()) {
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
		if err = tx.SetGameStatus(ctx, gameID, target, s.now()); err != nil {
			return err
		}
		if target == "ended" {
			if err = tx.ExpireValidations(ctx, gameID, s.now()); err != nil {
				return err
			}
			cancelled, cancelErr := tx.CancelActiveReactions(ctx, gameID, s.now())
			if cancelErr != nil {
				return cancelErr
			}
			for _, challengeID := range cancelled {
				challenge, challengeErr := tx.LockReactionChallenge(ctx, challengeID)
				if challengeErr != nil {
					return challengeErr
				}
				if challengeErr = tx.RecordReactionEvent(ctx, challenge, models.GameEventReactionChallengeExpired, nil,
					map[string]any{"status": "cancelled"}); challengeErr != nil {
					return challengeErr
				}
			}
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
	if _, err := s.FinishDueGame(ctx, p.GameID); err != nil {
		return models.Completion{}, err
	}
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
			if status != "assigned" {
				return fmt.Errorf("%w : cette mission n’est plus active", models.ErrConflict)
			}
			if g.Status != "playing" || gameExpired(g, s.now()) {
				return models.ErrConflict
			}
			if g.Mode == models.ModeTreasureHunt && s.Vision != nil {
				proof, err := tx.ValidProof(ctx, assignmentID)
				if err != nil {
					return err
				}
				if proof == nil {
					return fmt.Errorf("%w : une preuve photo valide est requise", models.ErrConflict)
				}
			}
			if g.ValidationMode == "peer" {
				if err = tx.RequestValidation(ctx, g.ID, p.ID, assignmentID); err != nil {
					return err
				}
				payload, _ := json.Marshal(map[string]any{"assignment_id": assignmentID})
				playerID := p.ID
				if err = tx.RecordGameEvent(ctx, &models.GameEvent{GameID: g.ID, PlayerID: &playerID, Type: models.GameEventValidationRequested, Payload: payload}); err != nil {
					return err
				}
				result.ValidationPending = true
				result.Player, err = tx.Player(ctx, p.ID)
				if err == nil {
					result.Mission, err = tx.CurrentMission(ctx, p.ID)
				}
				return err
			}
			awardedPoints := points
			allowChaosTrigger := true
			if g.Mode == models.ModeChaos {
				effect, effectErr := s.chaosEngine().ApplyCompletion(
					ctx, tx, g.ID, p.ID, points,
				)
				if effectErr != nil {
					return effectErr
				}
				awardedPoints = effect.AwardedPoints
				allowChaosTrigger = !effect.HadBlockingEvent
			}
			if gameFinalMinute(g, s.now()) {
				awardedPoints *= 2
			}
			if err = tx.CompleteAssignment(ctx, assignmentID, awardedPoints, s.now()); err != nil {
				return err
			}
			if err = tx.AddScore(ctx, p.ID, awardedPoints); err != nil {
				return err
			}
			if err = tx.AssignMission(ctx, p.ID); err != nil {
				return err
			}
			payload, marshalErr := json.Marshal(struct {
				AssignmentID string `json:"assignment_id"`
				Points       int    `json:"points"`
			}{AssignmentID: assignmentID, Points: awardedPoints})
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
			if g.Mode == models.ModeChaos {
				if err = s.chaosEngine().MaybeTriggerEvent(
					ctx, tx, g.ID, allowChaosTrigger, chaosThreshold(g, s.now()),
				); err != nil {
					return err
				}
			}
			result.AwardedPoints = awardedPoints
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

func gameExpired(g models.Game, now time.Time) bool { return g.EndsAt != nil && !now.Before(*g.EndsAt) }
func gameFinalMinute(g models.Game, now time.Time) bool {
	return g.EndsAt != nil && now.Before(*g.EndsAt) && !now.Before(g.EndsAt.Add(-time.Minute))
}
func chaosThreshold(g models.Game, now time.Time) int {
	if g.EndsAt == nil || g.StartedAt == nil {
		return 3
	}
	remaining := g.EndsAt.Sub(now)
	if remaining <= time.Minute {
		return 1
	}
	if remaining <= time.Duration(g.DurationMinutes)*time.Minute/3 {
		return 2
	}
	return 3
}
