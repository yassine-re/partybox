package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"partybox/backend/internal/models"
	"partybox/backend/internal/repositories"
	"partybox/backend/internal/vision"
)

const MaxProofAttempts = 5

var (
	ErrProofLimit    = errors.New("limite de tentatives photo atteinte pour cette mission")
	ErrProofProvider = errors.New("l’analyse photo est indisponible, réessaie dans un instant")
)

type ProofStatus struct {
	Available         bool   `json:"available"`
	AssignmentID      string `json:"assignment_id,omitempty"`
	Attempts          int    `json:"attempts"`
	MaxAttempts       int    `json:"max_attempts"`
	RemainingAttempts int    `json:"remaining_attempts"`
	Accepted          bool   `json:"accepted"`
}

type ProofResponse struct {
	Proof      *models.MissionProof `json:"proof"`
	Completion *models.Completion   `json:"completion,omitempty"`
}

func (s *Service) MissionProofStatus(ctx context.Context, p models.Player) (ProofStatus, error) {
	status := ProofStatus{MaxAttempts: MaxProofAttempts, RemainingAttempts: MaxProofAttempts}
	g, err := s.Game(ctx, p.GameID, p)
	if err != nil {
		return status, err
	}
	status.Available = s.Vision != nil && g.Mode == models.ModeTreasureHunt
	mission, err := s.Repo.Mission(ctx, p)
	if err != nil || mission == nil {
		return status, err
	}
	a, err := s.Repo.ProofAssignment(ctx, p, mission.ID)
	if err != nil {
		return status, err
	}
	status.AssignmentID, status.Attempts = mission.ID, a.Attempts
	status.RemainingAttempts = max(0, MaxProofAttempts-a.Attempts)
	proof, err := s.Repo.ValidProof(ctx, mission.ID)
	status.Accepted = proof != nil
	return status, err
}

func checkProofAssignment(a repositories.ProofAssignment) error {
	if a.Mode != models.ModeTreasureHunt {
		return fmt.Errorf("%w : photo réservée à Treasure Hunt", models.ErrConflict)
	}
	if a.Status != "assigned" || a.GameStatus != "playing" {
		return fmt.Errorf("%w : cette mission n’est plus active", models.ErrConflict)
	}
	return nil
}

func (s *Service) SubmitMissionProof(ctx context.Context, p models.Player, id string, image []byte) (ProofResponse, error) {
	var response ProofResponse
	if len(id) != 36 {
		return response, models.ErrInvalid
	}
	a, err := s.Repo.ProofAssignment(ctx, p, id)
	if err != nil {
		return response, err
	}
	if a.Mode != models.ModeTreasureHunt {
		return response, fmt.Errorf("%w : photo réservée à Treasure Hunt", models.ErrConflict)
	}
	// A lost success response must not cost a second vision call. Complete still
	// rechecks ownership/state and returns zero points for an already completed ID.
	accepted, err := s.Repo.ValidProof(ctx, id)
	if err != nil {
		return response, err
	}
	if accepted != nil {
		return s.completeProof(ctx, p, id, accepted)
	}
	if err = checkProofAssignment(a); err != nil {
		return response, err
	}
	if s.Vision == nil {
		return response, fmt.Errorf("%w : validation photo désactivée, utilise la validation manuelle", models.ErrConflict)
	}
	mediaType, err := vision.CheckImage(image)
	if err != nil {
		return response, fmt.Errorf("%w : %s", models.ErrInvalid, err)
	}
	err = s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
		if _, err := tx.LockGame(ctx, p.GameID); err != nil {
			return err
		}
		current, err := tx.ProofAssignment(ctx, p, id)
		if err != nil {
			return err
		}
		accepted, err = tx.ValidProof(ctx, id)
		if err != nil || accepted != nil {
			return err
		}
		if err = checkProofAssignment(current); err != nil {
			return err
		}
		count, err := tx.ProofCount(ctx, id)
		if err != nil {
			return err
		}
		if max(count, current.Attempts) >= MaxProofAttempts {
			return ErrProofLimit
		}
		return tx.ReserveProofAttempt(ctx, id)
	})
	if err != nil {
		return response, err
	}
	if accepted != nil {
		return s.completeProof(ctx, p, id, accepted)
	}
	// No database transaction or connection is held during the provider call.
	result, err := s.Vision.Validate(ctx, vision.ValidationRequest{MissionText: a.MissionText, Image: image, MediaType: mediaType})
	if err != nil || vision.CheckResult(result) != nil {
		return response, ErrProofProvider
	}
	proof := &models.MissionProof{AssignmentID: id, Verdict: string(result.Verdict), Confidence: result.Confidence, Reason: result.Reason}
	err = s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
		if err := tx.InsertProof(ctx, proof); err != nil {
			return err
		}
		payload, err := json.Marshal(map[string]any{"assignment_id": id, "verdict": proof.Verdict, "confidence": proof.Confidence})
		if err != nil {
			return err
		}
		return tx.RecordGameEvent(ctx, &models.GameEvent{GameID: p.GameID, PlayerID: &p.ID, Type: models.GameEventMissionProofEvaluated, Payload: payload})
	})
	if err != nil {
		return response, err
	}
	if result.Verdict == vision.Valid {
		return s.completeProof(ctx, p, id, proof)
	}
	return ProofResponse{Proof: proof}, nil
}

func (s *Service) completeProof(ctx context.Context, p models.Player, id string, proof *models.MissionProof) (ProofResponse, error) {
	completion, err := s.Complete(ctx, p, id)
	if err != nil {
		return ProofResponse{Proof: proof}, err
	}
	return ProofResponse{Proof: proof, Completion: &completion}, nil
}
