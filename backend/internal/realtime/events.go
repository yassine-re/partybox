package realtime

import "time"

type EventType string

const (
	EventPlayerJoined     EventType = "player_joined"
	EventGameStarted      EventType = "game_started"
	EventMissionCompleted EventType = "mission_completed"
	EventGameEnded        EventType = "game_ended"
)

// Event deliberately carries no game state or mission data. It only tells a
// client which REST resource changed; authenticated REST remains authoritative.
type Event struct {
	Type       EventType `json:"type"`
	GameID     string    `json:"game_id"`
	PlayerID   string    `json:"player_id,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}

func NewEvent(eventType EventType, gameID, playerID string) Event {
	return Event{
		Type:       eventType,
		GameID:     gameID,
		PlayerID:   playerID,
		OccurredAt: time.Now().UTC(),
	}
}
