package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"partybox/backend/internal/models"
)

type DeviceCredential struct {
	TokenHash string
	Enabled   bool
}

func (r *Repository) DeviceCredential(ctx context.Context, boxID string) (DeviceCredential, error) {
	var credential DeviceCredential
	err := r.Pool.QueryRow(ctx, `SELECT token_hash,enabled FROM box_devices WHERE box_id=$1`, boxID).
		Scan(&credential.TokenHash, &credential.Enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return credential, models.ErrUnauthorized
	}
	return credential, normalize(err)
}

func (r *Repository) ProvisionDevice(ctx context.Context, boxID, tokenHash string) error {
	tag, err := r.Pool.Exec(ctx, `INSERT INTO box_devices(box_id,token_hash,enabled)
		VALUES($1,$2,true) ON CONFLICT(box_id) DO UPDATE SET
		token_hash=excluded.token_hash,enabled=true,updated_at=now()`, boxID, tokenHash)
	if err != nil {
		return normalize(err)
	}
	if tag.RowsAffected() != 1 {
		return models.ErrNotFound
	}
	return nil
}

func (r *Repository) Heartbeat(ctx context.Context, boxID, firmwareVersion string, now time.Time) error {
	tag, err := r.Pool.Exec(ctx, `UPDATE box_devices SET last_seen_at=$2,firmware_version=$3,updated_at=$2
		WHERE box_id=$1 AND enabled`, boxID, now, firmwareVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return models.ErrUnauthorized
	}
	return nil
}

func (r *Repository) DeviceCommands(ctx context.Context, boxID string, now time.Time) ([]models.DeviceCommand, error) {
	if _, err := r.Pool.Exec(ctx, `UPDATE device_commands SET status='expired'
		WHERE box_id=$1 AND status IN ('pending','acknowledged') AND expires_at <= $2`, boxID, now); err != nil {
		return nil, err
	}
	rows, err := r.Pool.Query(ctx, `SELECT id,box_id,challenge_id,type,payload,status,created_at,expires_at
		FROM device_commands WHERE box_id=$1 AND status IN ('pending','acknowledged') AND expires_at>$2
		ORDER BY created_at,id LIMIT 10`, boxID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	commands := []models.DeviceCommand{}
	for rows.Next() {
		var command models.DeviceCommand
		var payload []byte
		if err = rows.Scan(&command.ID, &command.BoxID, &command.ChallengeID, &command.Type,
			&payload, &command.Status, &command.CreatedAt, &command.ExpiresAt); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(payload, &command.Payload); err != nil {
			return nil, err
		}
		commands = append(commands, command)
	}
	return commands, rows.Err()
}

func (t *Transaction) AcknowledgeDeviceCommand(ctx context.Context, boxID, commandID string, now time.Time) (string, bool, error) {
	var gameID, commandStatus, challengeStatus string
	var expiresAt time.Time
	err := t.tx.QueryRow(ctx, `SELECT rc.game_id,dc.status,rc.status,dc.expires_at
		FROM device_commands dc JOIN reaction_challenges rc ON rc.id=dc.challenge_id
		WHERE dc.id=$1 AND dc.box_id=$2 FOR UPDATE OF dc,rc`, commandID, boxID).
		Scan(&gameID, &commandStatus, &challengeStatus, &expiresAt)
	if err != nil {
		return "", false, normalize(err)
	}
	if !expiresAt.After(now) {
		_, _ = t.tx.Exec(ctx, `UPDATE device_commands SET status='expired' WHERE id=$1`, commandID)
		return gameID, false, models.ErrConflict
	}
	if commandStatus == "completed" || commandStatus == "expired" {
		return gameID, false, models.ErrConflict
	}
	if commandStatus == "acknowledged" {
		return gameID, false, nil
	}
	if challengeStatus != "awaiting_device" && challengeStatus != "armed" {
		return gameID, false, models.ErrConflict
	}
	if _, err = t.tx.Exec(ctx, `UPDATE device_commands SET status='acknowledged',acknowledged_at=$2 WHERE id=$1`, commandID, now); err != nil {
		return "", false, err
	}
	if _, err = t.tx.Exec(ctx, `UPDATE reaction_challenges SET status='armed',armed_at=COALESCE(armed_at,$2)
		WHERE id=(SELECT challenge_id FROM device_commands WHERE id=$1) AND status='awaiting_device'`, commandID, now); err != nil {
		return "", false, err
	}
	return gameID, true, nil
}
