package chaos

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"partybox/backend/internal/models"
)

type Reader interface {
	ChaosState(context.Context, string) (State, error)
}

type Store interface {
	InitializeChaosState(context.Context, string) error
	LockChaosState(context.Context, string) (State, error)
	UpdateChaosState(context.Context, State) error
	Players(context.Context, string) ([]models.Player, error)
	CancelCurrentMissions(context.Context, string) ([]CancelledMission, error)
	AssignMission(context.Context, string) error
	RecordGameEvent(context.Context, *models.GameEvent) error
}

type Engine struct {
	selector Selector
}

type CompletionEffect struct {
	AwardedPoints    int
	HadBlockingEvent bool
}

func NewEngine(selector Selector) *Engine {
	if selector == nil {
		selector = randomSelector{}
	}
	return &Engine{selector: selector}
}

func (engine *Engine) Initialize(ctx context.Context, store Store, gameID string) error {
	return store.InitializeChaosState(ctx, gameID)
}

func (engine *Engine) GetState(ctx context.Context, reader Reader, gameID string) (Snapshot, error) {
	state, err := reader.ChaosState(ctx, gameID)
	if err != nil {
		return Snapshot{}, err
	}
	return state.Snapshot(), nil
}

// ApplyCompletion computes server-authoritative points and consumes an active
// duration effect. The caller owns the assignment and score mutations, in the
// same PostgreSQL transaction.
func (engine *Engine) ApplyCompletion(
	ctx context.Context,
	store Store,
	gameID, playerID string,
	basePoints int,
) (CompletionEffect, error) {
	result := CompletionEffect{AwardedPoints: basePoints}
	state, err := store.LockChaosState(ctx, gameID)
	if err != nil {
		return result, err
	}
	if state.EventType == nil {
		return result, nil
	}

	switch *state.EventType {
	case EventDoubleTrouble:
		if state.RemainingUses <= 0 {
			clearEvent(&state)
			return result, store.UpdateChaosState(ctx, state)
		}
		result.HadBlockingEvent = true
		result.AwardedPoints = basePoints * DoubleMultiplier
		state.RemainingUses--
		if err = recordConsumed(ctx, store, state, playerID, result.AwardedPoints); err != nil {
			return result, err
		}
		if state.RemainingUses == 0 {
			clearEvent(&state)
		}
		return result, store.UpdateChaosState(ctx, state)

	case EventBounty:
		if state.RemainingUses <= 0 || state.TargetPlayerID == nil {
			clearEvent(&state)
			return result, store.UpdateChaosState(ctx, state)
		}
		result.HadBlockingEvent = true
		if *state.TargetPlayerID != playerID {
			return result, nil
		}
		bonus := BountyBonus
		var payload struct {
			Bonus int `json:"bonus"`
		}
		if json.Unmarshal(state.Payload, &payload) == nil && payload.Bonus > 0 {
			bonus = payload.Bonus
		}
		result.AwardedPoints = basePoints + bonus
		state.RemainingUses--
		if err = recordConsumed(ctx, store, state, playerID, result.AwardedPoints); err != nil {
			return result, err
		}
		if state.RemainingUses == 0 {
			clearEvent(&state)
		}
		return result, store.UpdateChaosState(ctx, state)

	case EventMissionShuffle:
		// Shuffle is instantaneous. Keep it visible until the next completion,
		// then resume normal progress toward a future event.
		clearEvent(&state)
		return result, store.UpdateChaosState(ctx, state)
	default:
		return result, fmt.Errorf("unknown chaos event %q", *state.EventType)
	}
}

// MaybeTriggerEvent advances the global completion counter and starts one
// deterministic-to-test random event when the threshold is reached.
func (engine *Engine) MaybeTriggerEvent(
	ctx context.Context,
	store Store,
	gameID string,
	allowTrigger bool,
) error {
	state, err := store.LockChaosState(ctx, gameID)
	if err != nil {
		return err
	}
	state.CompletionsSinceEvent++
	if !allowTrigger || (state.EventType != nil && state.RemainingUses > 0) ||
		state.CompletionsSinceEvent < CompletionThreshold {
		return store.UpdateChaosState(ctx, state)
	}

	selected := engine.selector.ChooseEvent(eventTypes)
	if !validEventType(selected) {
		return fmt.Errorf("invalid chaos event selected: %q", selected)
	}
	now := time.Now().UTC()
	state.EventType = &selected
	state.TargetPlayerID = nil
	state.TargetPlayerName = nil
	state.RemainingUses = 0
	state.CompletionsSinceEvent = 0
	state.EventSequence++
	state.StartedAt = &now

	triggerPayload := map[string]any{"event": selected}
	switch selected {
	case EventDoubleTrouble:
		state.RemainingUses = DoubleTroubleUses
		triggerPayload["remaining_uses"] = DoubleTroubleUses
		triggerPayload["multiplier"] = DoubleMultiplier
	case EventBounty:
		players, playersErr := store.Players(ctx, gameID)
		if playersErr != nil {
			return playersErr
		}
		if len(players) == 0 {
			return fmt.Errorf("cannot target a bounty without players")
		}
		target := engine.selector.ChoosePlayer(players)
		if !containsPlayer(players, target.ID) {
			return fmt.Errorf("invalid bounty target %q", target.ID)
		}
		state.TargetPlayerID = &target.ID
		state.TargetPlayerName = &target.Name
		state.RemainingUses = 1
		triggerPayload["target_player_id"] = target.ID
		triggerPayload["bonus"] = BountyBonus
		triggerPayload["remaining_uses"] = 1
	case EventMissionShuffle:
		triggerPayload["remaining_uses"] = 0
	}
	state.Payload, err = json.Marshal(triggerPayload)
	if err != nil {
		return err
	}
	if err = recordEvent(ctx, store, gameID, nil, models.GameEventChaosTriggered, triggerPayload); err != nil {
		return err
	}

	if selected == EventMissionShuffle {
		if err = engine.shuffleMissions(ctx, store, gameID, state.EventSequence); err != nil {
			return err
		}
		if err = recordEvent(ctx, store, gameID, nil, models.GameEventChaosConsumed, map[string]any{
			"event": selected, "remaining_uses": 0,
		}); err != nil {
			return err
		}
	}
	return store.UpdateChaosState(ctx, state)
}

func (engine *Engine) shuffleMissions(ctx context.Context, store Store, gameID string, sequence int) error {
	players, err := store.Players(ctx, gameID)
	if err != nil {
		return err
	}
	cancelled, err := store.CancelCurrentMissions(ctx, gameID)
	if err != nil {
		return err
	}
	for _, assignment := range cancelled {
		playerID := assignment.PlayerID
		if err = recordEvent(ctx, store, gameID, &playerID, models.GameEventMissionCancelled, map[string]any{
			"assignment_id": assignment.AssignmentID,
			"event":         EventMissionShuffle,
			"sequence":      sequence,
		}); err != nil {
			return err
		}
	}
	for _, player := range players {
		if err = store.AssignMission(ctx, player.ID); err != nil {
			return err
		}
	}
	return nil
}

func recordConsumed(ctx context.Context, store Store, state State, playerID string, awardedPoints int) error {
	return recordEvent(ctx, store, state.GameID, &playerID, models.GameEventChaosConsumed, map[string]any{
		"event":          *state.EventType,
		"remaining_uses": state.RemainingUses,
		"awarded_points": awardedPoints,
	})
}

func recordEvent(
	ctx context.Context,
	store Store,
	gameID string,
	playerID *string,
	eventType models.GameEventType,
	value any,
) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return store.RecordGameEvent(ctx, &models.GameEvent{
		GameID: gameID, PlayerID: playerID, Type: eventType, Payload: payload,
	})
}

func clearEvent(state *State) {
	state.EventType = nil
	state.TargetPlayerID = nil
	state.TargetPlayerName = nil
	state.RemainingUses = 0
	state.Payload = json.RawMessage(`{}`)
	state.StartedAt = nil
}

func containsPlayer(players []models.Player, playerID string) bool {
	for _, player := range players {
		if player.ID == playerID {
			return true
		}
	}
	return false
}
