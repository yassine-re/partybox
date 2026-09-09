package models

import (
	"encoding/json"
	"time"
)

type GameEventType string

const (
	GameEventPlayerJoined               GameEventType = "player_joined"
	GameEventGameStarted                GameEventType = "game_started"
	GameEventMissionAssigned            GameEventType = "mission_assigned"
	GameEventMissionCompleted           GameEventType = "mission_completed"
	GameEventGameEnded                  GameEventType = "game_ended"
	GameEventAIMissionsGenerated        GameEventType = "ai_missions_generated"
	GameEventMissionProofEvaluated      GameEventType = "mission_proof_evaluated"
	GameEventChaosTriggered             GameEventType = "chaos_event_triggered"
	GameEventChaosConsumed              GameEventType = "chaos_event_consumed"
	GameEventMissionCancelled           GameEventType = "mission_cancelled"
	GameEventMissionFeedbackSubmitted   GameEventType = "mission_feedback_submitted"
	GameEventReactionChallengeScheduled GameEventType = "reaction_challenge_scheduled"
	GameEventReactionChallengeAssigned  GameEventType = "reaction_challenge_assigned"
	GameEventReactionChallengeStarted   GameEventType = "reaction_challenge_started"
	GameEventReactionChallengeResolved  GameEventType = "reaction_challenge_resolved"
	GameEventReactionChallengeExpired   GameEventType = "reaction_challenge_expired"
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
