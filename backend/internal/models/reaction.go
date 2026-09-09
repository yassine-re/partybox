package models

import "time"

type Device struct {
	BoxID           string     `json:"box_id"`
	Enabled         bool       `json:"enabled"`
	LastSeenAt      *time.Time `json:"last_seen_at"`
	FirmwareVersion *string    `json:"firmware_version"`
}

type DeviceCommand struct {
	ID          string         `json:"id"`
	BoxID       string         `json:"box_id"`
	ChallengeID string         `json:"challenge_id"`
	Type        string         `json:"type"`
	Payload     map[string]any `json:"payload"`
	Status      string         `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	ExpiresAt   time.Time      `json:"expires_at"`
}

type ReactionChallenge struct {
	ID                   string     `json:"id"`
	GameID               string     `json:"game_id"`
	BoxID                string     `json:"box_id"`
	Kind                 string     `json:"kind"`
	Status               string     `json:"status"`
	ButtonS2PlayerID     *string    `json:"button_s2_player_id"`
	ButtonS2PlayerName   *string    `json:"button_s2_player_name"`
	ButtonS3PlayerID     *string    `json:"button_s3_player_id"`
	ButtonS3PlayerName   *string    `json:"button_s3_player_name"`
	DelayMS              int        `json:"delay_ms"`
	WinnerPlayerID       *string    `json:"winner_player_id"`
	WinnerPlayerName     *string    `json:"winner_player_name"`
	FalseStartPlayerID   *string    `json:"false_start_player_id"`
	FalseStartPlayerName *string    `json:"false_start_player_name"`
	ReactionMS           *int       `json:"reaction_ms"`
	AwardedPoints        int        `json:"awarded_points"`
	ResultEventID        *string    `json:"-"`
	ScheduledAt          time.Time  `json:"scheduled_at"`
	AssignedAt           *time.Time `json:"assigned_at"`
	ArmedAt              *time.Time `json:"armed_at"`
	ResolvedAt           *time.Time `json:"resolved_at"`
	ExpiresAt            time.Time  `json:"expires_at"`
}

type ReactionState struct {
	Enabled      bool               `json:"enabled"`
	DeviceOnline bool               `json:"device_online"`
	Challenge    *ReactionChallenge `json:"challenge"`
}

type ReactionResult struct {
	Challenge        ReactionChallenge `json:"challenge"`
	AlreadyProcessed bool              `json:"already_processed"`
}
