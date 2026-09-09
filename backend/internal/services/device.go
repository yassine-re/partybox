package services

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"partybox/backend/internal/deviceauth"
	"partybox/backend/internal/models"
	"partybox/backend/internal/repositories"
)

type HeartbeatInput struct {
	FirmwareVersion string `json:"firmware_version"`
	UptimeMS        uint64 `json:"uptime_ms"`
	WiFiRSSI        int    `json:"wifi_rssi"`
}

func (s *Service) AuthenticateDevice(ctx context.Context, boxID, token string) error {
	credential, err := s.Repo.DeviceCredential(ctx, boxID)
	if err != nil || !credential.Enabled || !deviceauth.ValidToken(token, credential.TokenHash) {
		return models.ErrUnauthorized
	}
	return nil
}

func (s *Service) DeviceHeartbeat(ctx context.Context, boxID string, input HeartbeatInput) (time.Time, error) {
	input.FirmwareVersion = strings.TrimSpace(input.FirmwareVersion)
	if !utf8.ValidString(input.FirmwareVersion) || len(input.FirmwareVersion) < 1 || len(input.FirmwareVersion) > 64 ||
		input.WiFiRSSI < -127 || input.WiFiRSSI > 0 {
		return time.Time{}, models.ErrInvalid
	}
	now := s.now()
	return now, s.Repo.Heartbeat(ctx, boxID, input.FirmwareVersion, now)
}

func (s *Service) DeviceCommands(ctx context.Context, boxID string) ([]models.DeviceCommand, error) {
	return s.Repo.DeviceCommands(ctx, boxID, s.now())
}

func (s *Service) AcknowledgeDeviceCommand(ctx context.Context, boxID, commandID string) (string, bool, error) {
	if len(commandID) != 36 {
		return "", false, models.ErrInvalid
	}
	var gameID string
	var changed bool
	err := s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
		var err error
		gameID, changed, err = tx.AcknowledgeDeviceCommand(ctx, boxID, commandID, s.now())
		if err != nil || !changed {
			return err
		}
		challenge, err := tx.LockReactionChallengeByCommand(ctx, commandID)
		if err != nil {
			return err
		}
		return tx.RecordReactionEvent(ctx, challenge, models.GameEventReactionChallengeStarted, nil, nil)
	})
	return gameID, changed, err
}

func validateBoxID(boxID string) error {
	if boxID == "" || len(boxID) > 40 {
		return fmt.Errorf("%w : box_id invalide", models.ErrInvalid)
	}
	for _, r := range boxID {
		if !(r == '-' || r == '_' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
			return fmt.Errorf("%w : box_id invalide", models.ErrInvalid)
		}
	}
	return nil
}

func (s *Service) ProvisionDevice(ctx context.Context, boxID, token string) error {
	if err := validateBoxID(boxID); err != nil {
		return err
	}
	if !deviceauth.ValidToken(token, deviceauth.HashToken(token)) {
		return fmt.Errorf("%w : le token doit contenir 32 octets aléatoires encodés en base64url", models.ErrInvalid)
	}
	return s.Repo.ProvisionDevice(ctx, boxID, deviceauth.HashToken(token))
}
