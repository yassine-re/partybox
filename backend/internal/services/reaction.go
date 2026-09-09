package services

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"partybox/backend/internal/models"
	"partybox/backend/internal/repositories"
)

const (
	ReactionButtonS2 = "S2"
	ReactionButtonS3 = "S3"
	ReactionTimeout  = 5 * time.Second
)

type ReactionConfig struct {
	Enabled           bool
	MinInterval       time.Duration
	MaxInterval       time.Duration
	AssignmentTimeout time.Duration
	ResultTimeout     time.Duration
	DeviceOnline      time.Duration
}

func DefaultReactionConfig() ReactionConfig {
	return ReactionConfig{
		Enabled: true, MinInterval: 45 * time.Second, MaxInterval: 90 * time.Second,
		AssignmentTimeout: 45 * time.Second, ResultTimeout: 15 * time.Second,
		DeviceOnline: 10 * time.Second,
	}
}

func ParseReactionConfig(getenv func(string) string) (ReactionConfig, error) {
	config := DefaultReactionConfig()
	values := []struct {
		key    string
		target *time.Duration
		min    int
		max    int
	}{
		{"REACTION_MIN_INTERVAL_SECONDS", &config.MinInterval, 5, 86400},
		{"REACTION_MAX_INTERVAL_SECONDS", &config.MaxInterval, 6, 86400},
		{"REACTION_ASSIGNMENT_TIMEOUT_SECONDS", &config.AssignmentTimeout, 5, 600},
		{"REACTION_RESULT_TIMEOUT_SECONDS", &config.ResultTimeout, 5, 120},
		{"DEVICE_ONLINE_TIMEOUT_SECONDS", &config.DeviceOnline, 3, 300},
	}
	if raw := strings.TrimSpace(getenv("REACTION_ENABLED")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return config, fmt.Errorf("REACTION_ENABLED: %w", err)
		}
		config.Enabled = value
	}
	for _, value := range values {
		raw := strings.TrimSpace(getenv(value.key))
		if raw == "" {
			continue
		}
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds < value.min || seconds > value.max {
			return config, fmt.Errorf("%s doit être compris entre %d et %d secondes", value.key, value.min, value.max)
		}
		*value.target = time.Duration(seconds) * time.Second
	}
	if config.MinInterval >= config.MaxInterval {
		return config, errors.New("REACTION_MIN_INTERVAL_SECONDS doit être inférieur à REACTION_MAX_INTERVAL_SECONDS")
	}
	return config, nil
}

type ReactionAssignment struct {
	ButtonS2PlayerID *string `json:"button_s2_player_id"`
	ButtonS3PlayerID *string `json:"button_s3_player_id"`
}

type DeviceReactionResult struct {
	EventID     string `json:"event_id"`
	ChallengeID string `json:"challenge_id"`
	CommandID   string `json:"command_id"`
	Button      string `json:"button"`
	ReactionMS  *int   `json:"reaction_ms"`
	FalseStart  bool   `json:"false_start"`
	Timeout     bool   `json:"timeout"`
}

type ReactionChange struct {
	GameID   string
	Resolved bool
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) randomIntN(limit int) int {
	if s.RandomIntN != nil {
		return s.RandomIntN(limit)
	}
	return rand.IntN(limit)
}

func (s *Service) nextReactionAt(now time.Time) time.Time {
	span := int((s.Reaction.MaxInterval - s.Reaction.MinInterval) / time.Second)
	return now.Add(s.Reaction.MinInterval + time.Duration(s.randomIntN(span+1))*time.Second)
}

func (s *Service) ReactionState(ctx context.Context, gameID string, player models.Player) (models.ReactionState, error) {
	if player.GameID != gameID {
		return models.ReactionState{}, models.ErrForbidden
	}
	now := s.now()
	return s.Repo.ReactionState(ctx, gameID, now.Add(-s.Reaction.DeviceOnline), s.Reaction.Enabled)
}

func (s *Service) TickReactions(ctx context.Context) ([]ReactionChange, error) {
	if !s.Reaction.Enabled {
		return nil, nil
	}
	now := s.now()
	missing, err := s.Repo.GamesMissingReactionSchedule(ctx)
	if err != nil {
		return nil, err
	}
	for _, gameID := range missing {
		if err = s.Repo.EnsureReactionSchedule(ctx, gameID, s.nextReactionAt(now)); err != nil {
			return nil, err
		}
	}
	changes := []ReactionChange{}
	expirable, err := s.Repo.ExpirableReactionChallenges(ctx, now)
	if err != nil {
		return nil, err
	}
	for _, challengeID := range expirable {
		var gameID, status string
		var changed bool
		err = s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
			var txErr error
			gameID, status, changed, txErr = tx.ExpireReaction(ctx, challengeID, now, s.nextReactionAt(now))
			if txErr != nil || !changed {
				return txErr
			}
			challenge, txErr := tx.LockReactionChallenge(ctx, challengeID)
			if txErr != nil {
				return txErr
			}
			return tx.RecordReactionEvent(ctx, challenge, models.GameEventReactionChallengeExpired, nil,
				map[string]any{"status": status})
		})
		if err != nil {
			return changes, err
		}
		if changed {
			changes = append(changes, ReactionChange{GameID: gameID})
		}
	}
	due, err := s.Repo.DueReactionGames(ctx, now, now.Add(-s.Reaction.DeviceOnline))
	if err != nil {
		return changes, err
	}
	for _, gameID := range due {
		var scheduled bool
		err = s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
			game, txErr := tx.LockGame(ctx, gameID)
			if txErr != nil {
				return txErr
			}
			if game.Status != "playing" {
				return nil
			}
			next, txErr := tx.LockReactionState(ctx, gameID)
			if txErr != nil || next == nil || next.After(now) {
				return txErr
			}
			online, txErr := tx.DeviceOnline(ctx, game.BoxID, now.Add(-s.Reaction.DeviceOnline))
			if txErr != nil || !online {
				return txErr
			}
			active, txErr := tx.ActiveReactionExists(ctx, gameID)
			if txErr != nil || active {
				return txErr
			}
			players, txErr := tx.Players(ctx, gameID)
			if txErr != nil || len(players) == 0 {
				return txErr
			}
			kind := "solo"
			if len(players) >= 2 && s.randomIntN(2) == 1 {
				kind = "duel"
			}
			delayMS := 2000 + s.randomIntN(4001)
			challenge, txErr := tx.InsertReactionChallenge(ctx, gameID, game.BoxID, kind, delayMS,
				now, now.Add(s.Reaction.AssignmentTimeout))
			if txErr != nil {
				return txErr
			}
			if txErr = tx.RecordReactionEvent(ctx, challenge, models.GameEventReactionChallengeScheduled, nil, nil); txErr != nil {
				return txErr
			}
			scheduled = true
			return nil
		})
		if err != nil {
			if errors.Is(err, models.ErrConflict) {
				continue
			}
			return changes, err
		}
		if scheduled {
			changes = append(changes, ReactionChange{GameID: gameID})
		}
	}
	return changes, nil
}

func (s *Service) AssignReaction(ctx context.Context, gameID, challengeID string, host models.Player, assignment ReactionAssignment) (models.ReactionChallenge, error) {
	if host.GameID != gameID || !host.IsHost {
		return models.ReactionChallenge{}, models.ErrForbidden
	}
	if len(challengeID) != 36 {
		return models.ReactionChallenge{}, models.ErrInvalid
	}
	now := s.now()
	var result models.ReactionChallenge
	err := s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
		game, err := tx.LockGame(ctx, gameID)
		if err != nil {
			return err
		}
		if game.Status != "playing" {
			return models.ErrConflict
		}
		challenge, err := tx.LockReactionChallenge(ctx, challengeID)
		if err != nil {
			return err
		}
		if challenge.GameID != gameID {
			return models.ErrNotFound
		}
		if challenge.Status != "awaiting_assignment" || !challenge.ExpiresAt.After(now) {
			return models.ErrConflict
		}
		online, err := tx.DeviceOnline(ctx, game.BoxID, now.Add(-s.Reaction.DeviceOnline))
		if err != nil {
			return err
		}
		if !online {
			return fmt.Errorf("%w : PartyBox physique hors ligne", models.ErrConflict)
		}
		if err = validateReactionAssignment(ctx, tx, challenge, assignment); err != nil {
			return err
		}
		expires := now.Add(time.Duration(challenge.DelayMS)*time.Millisecond + s.Reaction.ResultTimeout)
		if _, err = tx.AssignReaction(ctx, challenge, assignment.ButtonS2PlayerID, assignment.ButtonS3PlayerID, now, expires); err != nil {
			return err
		}
		extra := map[string]any{"button_s2_player_id": assignment.ButtonS2PlayerID, "button_s3_player_id": assignment.ButtonS3PlayerID}
		if err = tx.RecordReactionEvent(ctx, challenge, models.GameEventReactionChallengeAssigned, &host.ID, extra); err != nil {
			return err
		}
		result, err = tx.LockReactionChallenge(ctx, challengeID)
		return err
	})
	return result, err
}

func validateReactionAssignment(ctx context.Context, tx *repositories.Transaction, challenge models.ReactionChallenge, assignment ReactionAssignment) error {
	ids := []*string{assignment.ButtonS2PlayerID, assignment.ButtonS3PlayerID}
	for _, id := range ids {
		if id == nil {
			continue
		}
		if len(*id) != 36 {
			return models.ErrInvalid
		}
		belongs, err := tx.PlayerBelongsToGame(ctx, *id, challenge.GameID)
		if err != nil {
			return err
		}
		if !belongs {
			return models.ErrForbidden
		}
	}
	if challenge.Kind == "solo" {
		if (assignment.ButtonS2PlayerID == nil) == (assignment.ButtonS3PlayerID == nil) {
			return fmt.Errorf("%w : choisir exactement un joueur sur S2 ou S3", models.ErrInvalid)
		}
		return nil
	}
	if assignment.ButtonS2PlayerID == nil || assignment.ButtonS3PlayerID == nil ||
		*assignment.ButtonS2PlayerID == *assignment.ButtonS3PlayerID {
		return fmt.Errorf("%w : S2 et S3 doivent être attribués à deux joueurs distincts", models.ErrInvalid)
	}
	return nil
}

func SoloReactionPoints(reactionMS int, falseStart, timeout bool) int {
	if falseStart || timeout || reactionMS < 0 {
		return 0
	}
	switch {
	case reactionMS < 200:
		return 150
	case reactionMS < 300:
		return 100
	case reactionMS < 400:
		return 75
	case reactionMS < 600:
		return 50
	default:
		return 25
	}
}

func validEventID(value string) bool {
	if !utf8.ValidString(value) || len(value) < 1 || len(value) > 160 {
		return false
	}
	for _, char := range value {
		if unicode.IsControl(char) || unicode.IsSpace(char) {
			return false
		}
	}
	return true
}

func assignedPlayer(challenge models.ReactionChallenge, button string) *string {
	switch button {
	case ReactionButtonS2:
		return challenge.ButtonS2PlayerID
	case ReactionButtonS3:
		return challenge.ButtonS3PlayerID
	default:
		return nil
	}
}

func opponent(challenge models.ReactionChallenge, playerID string) *string {
	if challenge.ButtonS2PlayerID != nil && *challenge.ButtonS2PlayerID != playerID {
		return challenge.ButtonS2PlayerID
	}
	if challenge.ButtonS3PlayerID != nil && *challenge.ButtonS3PlayerID != playerID {
		return challenge.ButtonS3PlayerID
	}
	return nil
}

func (s *Service) SubmitReactionResult(ctx context.Context, boxID string, input DeviceReactionResult) (models.ReactionResult, error) {
	if len(input.ChallengeID) != 36 || len(input.CommandID) != 36 || !validEventID(input.EventID) {
		return models.ReactionResult{}, models.ErrInvalid
	}
	gameID, challengeBoxID, err := s.Repo.ReactionChallengeIdentity(ctx, input.ChallengeID)
	if err != nil {
		return models.ReactionResult{}, err
	}
	if challengeBoxID != boxID {
		return models.ReactionResult{}, models.ErrForbidden
	}
	now := s.now()
	var result models.ReactionResult
	err = s.Repo.Transaction(ctx, func(tx *repositories.Transaction) error {
		game, err := tx.LockGame(ctx, gameID)
		if err != nil {
			return err
		}
		challenge, err := tx.LockReactionChallenge(ctx, input.ChallengeID)
		if err != nil {
			return err
		}
		if challenge.Status == "resolved" {
			if challenge.ResultEventID != nil && *challenge.ResultEventID == input.EventID {
				result = models.ReactionResult{Challenge: challenge, AlreadyProcessed: true}
				return nil
			}
			return models.ErrConflict
		}
		if game.Status != "playing" || challenge.Status != "armed" {
			return models.ErrConflict
		}
		matches, err := tx.CommandMatchesChallenge(ctx, boxID, input.CommandID, input.ChallengeID)
		if err != nil {
			return err
		}
		if !matches {
			return models.ErrForbidden
		}
		var winnerID, falseStartID *string
		var reactionMS *int
		points := 0
		if input.Timeout {
			if input.FalseStart || input.Button != "" || input.ReactionMS != nil {
				return models.ErrInvalid
			}
		} else {
			pressed := assignedPlayer(challenge, input.Button)
			if pressed == nil {
				return fmt.Errorf("%w : bouton non attribué", models.ErrInvalid)
			}
			if input.FalseStart {
				if input.ReactionMS != nil {
					return models.ErrInvalid
				}
				falseStartID = pressed
				if challenge.Kind == "duel" {
					winnerID = opponent(challenge, *pressed)
					points = 100
				}
			} else {
				if input.ReactionMS == nil || *input.ReactionMS < 0 {
					return models.ErrInvalid
				}
				reactionMS = input.ReactionMS
				if *input.ReactionMS < int(ReactionTimeout/time.Millisecond) {
					winnerID = pressed
					if challenge.Kind == "duel" {
						points = 100
					} else {
						points = SoloReactionPoints(*input.ReactionMS, false, false)
					}
				}
			}
		}
		if winnerID == nil {
			points = 0
		}
		if err = tx.ResolveReaction(ctx, challenge.ID, input.EventID, winnerID, falseStartID, reactionMS, points, now); err != nil {
			return err
		}
		if err = tx.ReplanReaction(ctx, game.ID, now, s.nextReactionAt(now)); err != nil {
			return err
		}
		extra := map[string]any{
			"winner_player_id": winnerID, "reaction_ms": reactionMS,
			"false_start": falseStartID != nil, "false_start_player_id": falseStartID,
			"awarded_points": points,
		}
		if err = tx.RecordReactionEvent(ctx, challenge, models.GameEventReactionChallengeResolved, winnerID, extra); err != nil {
			return err
		}
		resolved, err := tx.LockReactionChallenge(ctx, challenge.ID)
		if err != nil {
			return err
		}
		result.Challenge = resolved
		return nil
	})
	if err != nil {
		return models.ReactionResult{}, err
	}
	return result, nil
}
