package realtime

import "time"

type EventType string

const (
	EventPlayerJoined              EventType = "player_joined"
	EventGameStarted               EventType = "game_started"
	EventMissionCompleted          EventType = "mission_completed"
	EventGameEnded                 EventType = "game_ended"
	EventFiveMinutes               EventType = "five_minutes_remaining"
	EventFinalMinute               EventType = "final_minute"
	EventValidationRequested       EventType = "validation_requested"
	EventValidationResolved        EventType = "validation_resolved"
	EventChaosChanged              EventType = "chaos_changed"
	EventLeaderboardChanged        EventType = "leaderboard_changed"
	EventReactionChallengeChanged  EventType = "reaction_challenge_changed"
	EventReactionChallengeResolved EventType = "reaction_challenge_resolved"
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
