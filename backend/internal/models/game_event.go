package models

import (
	"encoding/json"
	"time"
)

type GameEventType string

const (
	GameEventPlayerJoined     GameEventType = "player_joined"
	GameEventGameStarted      GameEventType = "game_started"
	GameEventMissionAssigned  GameEventType = "mission_assigned"
	GameEventMissionCompleted GameEventType = "mission_completed"
	GameEventGameEnded        GameEventType = "game_ended"
)

// GameEvent is persisted for history and analytics. It is distinct from the
// ephemeral realtime.Event sent to connected browsers.
type GameEvent struct {
	ID        string          `json:"id"`
	GameID    string          `json:"game_id"`
	PlayerID  *string         `json:"player_id"`
	Type      GameEventType   `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}
