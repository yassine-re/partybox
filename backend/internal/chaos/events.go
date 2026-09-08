package chaos

import (
	"math/rand/v2"

	"partybox/backend/internal/models"
)

type EventType string

const (
	EventDoubleTrouble  EventType = "double_trouble"
	EventBounty         EventType = "bounty"
	EventMissionShuffle EventType = "mission_shuffle"

	DoubleTroubleUses = 3
	DoubleMultiplier  = 2
	BountyBonus       = 100
)

var eventTypes = []EventType{EventDoubleTrouble, EventBounty, EventMissionShuffle}

// Selector is the only source of randomness used by the engine. Tests can
// inject a deterministic implementation without changing gameplay code.
type Selector interface {
	ChooseEvent([]EventType) EventType
	ChoosePlayer([]models.Player) models.Player
}

type randomSelector struct{}

func (randomSelector) ChooseEvent(events []EventType) EventType {
	return events[rand.IntN(len(events))]
}

func (randomSelector) ChoosePlayer(players []models.Player) models.Player {
	return players[rand.IntN(len(players))]
}

func validEventType(value EventType) bool {
	for _, eventType := range eventTypes {
		if value == eventType {
			return true
		}
	}
	return false
}
