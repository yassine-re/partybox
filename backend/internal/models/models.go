package models

import (
	"errors"
	"time"
)

var (
	ErrNotFound     = errors.New("ressource introuvable")
	ErrUnauthorized = errors.New("session invalide ou expirée")
	ErrForbidden    = errors.New("action réservée à l’hôte de cette partie")
	ErrConflict     = errors.New("action incompatible avec l’état actuel")
	ErrInvalid      = errors.New("données invalides")
)

type Box struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ActiveGame *Game  `json:"active_game"`
}

type Game struct {
	ID                    string     `json:"id"`
	BoxID                 string     `json:"box_id"`
	Name                  string     `json:"name"`
	Mode                  GameMode   `json:"mode"`
	Status                string     `json:"status"`
	CreatedAt             time.Time  `json:"created_at"`
	StartedAt             *time.Time `json:"started_at"`
	EndedAt               *time.Time `json:"ended_at"`
	DurationMinutes       int        `json:"duration_minutes"`
	EndsAt                *time.Time `json:"ends_at"`
	FinishedAt            *time.Time `json:"finished_at"`
	LeaderboardVisibility string     `json:"leaderboard_visibility"`
	ValidationMode        string     `json:"validation_mode"`
	Players               []Player   `json:"players,omitempty"`
}

type Player struct {
	ID                string `json:"id"`
	GameID            string `json:"game_id"`
	Name              string `json:"name"`
	Score             int    `json:"score,omitempty"`
	IsHost            bool   `json:"is_host"`
	CompletedMissions int    `json:"completed_missions,omitempty"`
}

type Mission struct {
	ID         string `json:"id"` // Assignment ID, used to make completion retry-safe.
	MissionID  int    `json:"mission_id"`
	Text       string `json:"text"`
	Points     int    `json:"points"`
	Category   string `json:"category"`
	Difficulty int    `json:"difficulty"`
}

type MissionInput struct {
	Text       string `json:"text"`
	Points     int    `json:"points"`
	Category   string `json:"category"`
	Difficulty int    `json:"difficulty"`
}

type Session struct {
	Token  string `json:"token"`
	Player Player `json:"player"`
	Game   Game   `json:"game"`
}

type Completion struct {
	AwardedPoints     int      `json:"awarded_points"`
	AlreadyCompleted  bool     `json:"already_completed"`
	Player            Player   `json:"player"`
	Mission           *Mission `json:"mission"`
	ValidationPending bool     `json:"validation_pending,omitempty"`
}

type MissionSkip struct {
	Player  Player  `json:"player"`
	Mission Mission `json:"mission"`
}

type GameOptions struct {
	DurationMinutes       int    `json:"duration_minutes"`
	LeaderboardVisibility string `json:"leaderboard_visibility"`
	ValidationMode        string `json:"validation_mode"`
}

type ValidationRequest struct {
	ID           string    `json:"id"`
	AssignmentID string    `json:"assignment_id"`
	PlayerID     string    `json:"player_id"`
	PlayerName   string    `json:"player_name"`
	MissionText  string    `json:"mission_text"`
	RequestedAt  time.Time `json:"requested_at"`
}

type FinalMission struct {
	Text          string     `json:"text"`
	Status        string     `json:"status"`
	Points        int        `json:"points"`
	AwardedPoints int        `json:"awarded_points"`
	CompletedAt   *time.Time `json:"completed_at"`
	ProofURL      *string    `json:"proof_url,omitempty"`
}

type FinalPlayer struct {
	Player   Player         `json:"player"`
	Missions []FinalMission `json:"missions"`
}

type FinalRecap struct {
	Players     []FinalPlayer `json:"players"`
	ChaosEvents []GameEvent   `json:"chaos_events,omitempty"`
}
