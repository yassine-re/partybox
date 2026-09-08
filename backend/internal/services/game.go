package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"partybox/backend/internal/ai"
	"partybox/backend/internal/models"
	"partybox/backend/internal/repositories"
)

type Service struct {
	Repo *repositories.Repository
	AI   ai.MissionGenerator

	aiUsageMu sync.Mutex
	aiUsage   map[string]aiGenerationUsage
}

func cleanName(value string, max int) (string, error) {
	value = strings.TrimSpace(value)
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) < 1 || utf8.RuneCountInString(value) > max {
		return "", fmt.Errorf("%w : le nom doit contenir entre 1 et %d caractères", models.ErrInvalid, max)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return "", models.ErrInvalid
		}
	}
	return value, nil
}

func TokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newToken() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes[:]), nil
}

func (s *Service) Create(ctx context.Context, boxID, gameName, playerName string, mode models.GameMode) (models.Session, error) {
	if _, ok := models.LookupGameMode(mode); !ok {
		return models.Session{}, fmt.Errorf("%w : mode de jeu non supporté", models.ErrInvalid)
	}
	gameName, err := cleanName(gameName, 60)
	if err != nil {
		return models.Session{}, err
	}
	playerName, err = cleanName(playerName, 24)
	if err != nil {
		return models.Session{}, err
	}
	token, err := newToken()
	if err != nil {
		return models.Session{}, err
	}
	result, err := s.create(ctx, boxID, gameName, playerName, TokenHash(token), mode)
	if err == nil {
		result.Token = token
	}
	return result, err
}

func (s *Service) Join(ctx context.Context, gameID, name string) (models.Session, error) {
	name, err := cleanName(name, 24)
	if err != nil {
		return models.Session{}, err
	}
	token, err := newToken()
	if err != nil {
		return models.Session{}, err
	}
	result, err := s.join(ctx, gameID, name, TokenHash(token))
	if err == nil {
		result.Token = token
	}
	return result, err
}

func (s *Service) Authenticate(ctx context.Context, token string) (models.Player, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != 32 {
		return models.Player{}, models.ErrUnauthorized
	}
	return s.Repo.Authenticate(ctx, TokenHash(token))
}

func (s *Service) Game(ctx context.Context, id string, p models.Player) (models.Game, error) {
	if p.GameID != id {
		return models.Game{}, models.ErrForbidden
	}
	return s.Repo.Game(ctx, id)
}

func (s *Service) Start(ctx context.Context, id string, p models.Player) error {
	if p.GameID != id || !p.IsHost {
		return models.ErrForbidden
	}
	if err := s.beginGameTransition(id); err != nil {
		return err
	}
	defer s.finishGameTransition(id)
	return s.transition(ctx, id, p, "playing")
}

func (s *Service) End(ctx context.Context, id string, p models.Player) error {
	if p.GameID != id || !p.IsHost {
		return models.ErrForbidden
	}
	if err := s.beginGameTransition(id); err != nil {
		return err
	}
	defer s.finishGameTransition(id)
	return s.transition(ctx, id, p, "ended")
}

func (s *Service) Complete(ctx context.Context, p models.Player, id string) (models.Completion, error) {
	if len(id) != 36 {
		return models.Completion{}, models.ErrInvalid
	}
	return s.complete(ctx, p, id)
}
