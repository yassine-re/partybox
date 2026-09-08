package ai

import (
	"context"

	"partybox/backend/internal/models"
)

type GenerationRequest struct {
	Mode        models.GameMode `json:"mode"`
	Vibe        string          `json:"vibe"`
	Intensity   int             `json:"intensity"`
	Context     string          `json:"context"`
	Count       int             `json:"count"`
	PlayerCount int             `json:"player_count"`
}

type GeneratedMission = models.MissionInput

type MissionGenerator interface {
	Generate(ctx context.Context, input GenerationRequest) ([]GeneratedMission, error)
}
