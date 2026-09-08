package chaos

import (
	"encoding/json"
	"time"
)

const CompletionThreshold = 3

type State struct {
	GameID                string
	EventType             *EventType
	TargetPlayerID        *string
	TargetPlayerName      *string
	RemainingUses         int
	CompletionsSinceEvent int
	Payload               json.RawMessage
	EventSequence         int
	StartedAt             *time.Time
	UpdatedAt             time.Time
}

type EventState struct {
	Type             EventType `json:"type"`
	RemainingUses    int       `json:"remaining_uses"`
	TargetPlayerID   *string   `json:"target_player_id"`
	TargetPlayerName *string   `json:"target_player_name"`
	Bonus            int       `json:"bonus,omitempty"`
}

type Progress struct {
	Completed int `json:"completed"`
	TriggerAt int `json:"trigger_at"`
}

type Snapshot struct {
	Active   bool        `json:"active"`
	Event    *EventState `json:"event"`
	Progress Progress    `json:"progress"`
	Sequence int         `json:"sequence"`
}

type CancelledMission struct {
	AssignmentID string
	PlayerID     string
}

func (state State) Snapshot() Snapshot {
	completed := min(state.CompletionsSinceEvent, CompletionThreshold)
	result := Snapshot{
		Progress: Progress{Completed: completed, TriggerAt: CompletionThreshold},
		Sequence: state.EventSequence,
	}
	if state.EventType == nil {
		return result
	}
	result.Active = true
	result.Event = &EventState{
		Type:             *state.EventType,
		RemainingUses:    state.RemainingUses,
		TargetPlayerID:   state.TargetPlayerID,
		TargetPlayerName: state.TargetPlayerName,
	}
	if *state.EventType == EventBounty {
		var payload struct {
			Bonus int `json:"bonus"`
		}
		if json.Unmarshal(state.Payload, &payload) == nil {
			result.Event.Bonus = payload.Bonus
		}
		if result.Event.Bonus == 0 {
			result.Event.Bonus = BountyBonus
		}
	}
	return result
}
