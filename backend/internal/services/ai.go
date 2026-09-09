package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"partybox/backend/internal/ai"
	"partybox/backend/internal/models"
	"partybox/backend/internal/repositories"
)

type AIGenerationOptions struct {
	Vibe      string `json:"vibe"`
	Intensity int    `json:"intensity"`
	Context   string `json:"context"`
	Count     int    `json:"count"`
}

type AIStatus struct {
	Available            bool `json:"available"`
	Count                int  `json:"count"`
	RemainingGenerations int  `json:"remaining_generations"`
}

const maxAIGenerationAttemptsPerGame = 5

type aiGenerationUsage struct {
	attempts    int
	generating  bool
	transitions int
}

func (s *Service) aiAttempts(gameID string, persisted int) int {
	s.aiUsageMu.Lock()
	defer s.aiUsageMu.Unlock()
	if s.aiUsage == nil {
		s.aiUsage = make(map[string]aiGenerationUsage)
	}
	usage := s.aiUsage[gameID]
	if usage.attempts < persisted {
		usage.attempts = persisted
		s.aiUsage[gameID] = usage
	}
	return usage.attempts
}

func (s *Service) beginAIGeneration(gameID string, persisted int) error {
	s.aiUsageMu.Lock()
	defer s.aiUsageMu.Unlock()
	if s.aiUsage == nil {
		s.aiUsage = make(map[string]aiGenerationUsage)
	}
	usage := s.aiUsage[gameID]
	if usage.attempts < persisted {
		usage.attempts = persisted
	}
	if usage.generating {
		return fmt.Errorf("%w : une génération IA est déjà en cours pour cette partie", models.ErrConflict)
	}
	if usage.transitions > 0 {
		return fmt.Errorf("%w : la partie est en cours de changement d’état", models.ErrConflict)
	}
	if usage.attempts >= maxAIGenerationAttemptsPerGame {
		return fmt.Errorf("%w : limite de %d générations IA atteinte pour cette partie", models.ErrConflict, maxAIGenerationAttemptsPerGame)
	}
	usage.attempts++
	usage.generating = true
	s.aiUsage[gameID] = usage
	return nil
}

func (s *Service) finishAIGeneration(gameID string) {
	s.aiUsageMu.Lock()
	defer s.aiUsageMu.Unlock()
	usage := s.aiUsage[gameID]
	usage.generating = false
	s.aiUsage[gameID] = usage
}

func (s *Service) beginGameTransition(gameID string) error {
	s.aiUsageMu.Lock()
	defer s.aiUsageMu.Unlock()
	if s.aiUsage == nil {
		s.aiUsage = make(map[string]aiGenerationUsage)
	}
	usage := s.aiUsage[gameID]
	if usage.generating {
		return fmt.Errorf("%w : attends la fin de la génération IA avant de lancer ou terminer la partie", models.ErrConflict)
	}
	usage.transitions++
	s.aiUsage[gameID] = usage
	return nil
}

func (s *Service) finishGameTransition(gameID string) {
	s.aiUsageMu.Lock()
	defer s.aiUsageMu.Unlock()
	usage := s.aiUsage[gameID]
	if usage.transitions > 0 {
		usage.transitions--
	}
	s.aiUsage[gameID] = usage
}

func (s *Service) AIMissionsStatus(ctx context.Context, gameID string, p models.Player) (AIStatus, error) {
	if p.GameID != gameID {
		return AIStatus{}, models.ErrForbidden
	}
	g, err := s.Repo.Game(ctx, gameID)
	if err != nil {
		return AIStatus{}, err
	}
	count, err := s.Repo.AIMissionsCount(ctx, gameID)
	if err != nil {
		return AIStatus{}, err
	}
	persistedAttempts, err := s.Repo.AIMissionGenerationCount(ctx, gameID)
	if err != nil {
		return AIStatus{}, err
	}
	attempts := s.aiAttempts(gameID, persistedAttempts)
	remaining := maxAIGenerationAttemptsPerGame - attempts
	if remaining < 0 {
		remaining = 0
	}
	return AIStatus{
		Available:            s.AI != nil && models.SupportsAIGeneration(g.Mode),
		Count:                count,
		RemainingGenerations: remaining,
	}, nil
}

func (s *Service) GenerateAIMissions(ctx context.Context, gameID string, p models.Player, opts AIGenerationOptions) (int, error) {
	if p.GameID != gameID || !p.IsHost {
		return 0, models.ErrForbidden
	}
	if s.AI == nil {
		return 0, fmt.Errorf("%w : la génération IA n’est pas configurée sur ce serveur", models.ErrConflict)
	}

	g, err := s.Repo.Game(ctx, gameID)
	if err != nil {
		return 0, err
	}
	if g.Status != "lobby" {
		return 0, fmt.Errorf("%w : la génération n’est possible que dans le lobby", models.ErrConflict)
	}
	if !models.SupportsAIGeneration(g.Mode) {
		return 0, fmt.Errorf("%w : mode de jeu non compatible avec la génération IA", models.ErrConflict)
	}

	// Validate inputs
	vibe := strings.ToLower(strings.TrimSpace(opts.Vibe))
	if vibe == "" {
		vibe = "fun"
	}
	if vibe != "chill" && vibe != "fun" && vibe != "chaos" {
		return 0, fmt.Errorf("%w : ambiance invalide (options : chill, fun, chaos)", models.ErrInvalid)
	}

	intensity := opts.Intensity
	if intensity == 0 {
		intensity = 7
	}
	if intensity < 1 || intensity > 10 {
		return 0, fmt.Errorf("%w : intensité invalide (%d), doit être entre 1 et 10", models.ErrInvalid, intensity)
	}

	gameContext := strings.TrimSpace(opts.Context)
	if utf8.RuneCountInString(gameContext) > 300 {
		return 0, fmt.Errorf("%w : contexte trop long (maximum 300 caractères)", models.ErrInvalid)
	}

	count := opts.Count
	if count <= 0 {
		count = 20
	}
	if count < 5 || count > 50 {
		return 0, fmt.Errorf("%w : nombre de missions demandé (%d) invalide (entre 5 et 50)", models.ErrInvalid, count)
	}
	persistedAttempts, err := s.Repo.AIMissionGenerationCount(ctx, gameID)
	if err != nil {
		return 0, err
	}
	if err := s.beginAIGeneration(gameID, persistedAttempts); err != nil {
		return 0, err
	}
	defer s.finishAIGeneration(gameID)

	// 1. Call AI generator outside of DB transaction
	genReq := ai.GenerationRequest{
		Mode:        g.Mode,
		Vibe:        vibe,
		Intensity:   intensity,
		Context:     gameContext,
		Count:       count,
		PlayerCount: len(g.Players),
	}

	missions, err := s.AI.Generate(ctx, genReq)
	if err != nil {
		return 0, err
	}

	// 2. Validate in memory before any DB modification
	if err := ai.ValidateMissions(missions, count); err != nil {
		return 0, err
	}

	// 3. Atomically replace the AI missions in a database transaction
	err = s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
		lockedGame, err := tx.LockGame(ctx, gameID)
		if err != nil {
			return err
		}
		if lockedGame.Status != "lobby" {
			return fmt.Errorf("%w : la partie a été lancée pendant la génération", models.ErrConflict)
		}

		if err := tx.ReplaceAIMissions(ctx, gameID, g.Mode, missions); err != nil {
			return err
		}

		payload, err := json.Marshal(map[string]any{
			"count":     len(missions),
			"mode":      g.Mode,
			"vibe":      vibe,
			"intensity": intensity,
		})
		if err != nil {
			return err
		}

		playerID := p.ID
		return tx.RecordGameEvent(ctx, &models.GameEvent{
			GameID:   gameID,
			PlayerID: &playerID,
			Type:     models.GameEventAIMissionsGenerated,
			Payload:  payload,
		})
	})
	if err != nil {
		return 0, err
	}

	return len(missions), nil
}
