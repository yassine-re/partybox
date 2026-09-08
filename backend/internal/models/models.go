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
	ID        string     `json:"id"`
	BoxID     string     `json:"box_id"`
	Name      string     `json:"name"`
	Mode      string     `json:"mode"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	StartedAt *time.Time `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
	Players   []Player   `json:"players,omitempty"`
}

type Player struct {
	ID                string `json:"id"`
	GameID            string `json:"game_id"`
	Name              string `json:"name"`
	Score             int    `json:"score"`
	IsHost            bool   `json:"is_host"`
	CompletedMissions int    `json:"completed_missions"`
}

type Mission struct {
	ID         string `json:"id"` // Assignment ID, used to make completion retry-safe.
	MissionID  int    `json:"mission_id"`
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
	AwardedPoints    int      `json:"awarded_points"`
	AlreadyCompleted bool     `json:"already_completed"`
	Player           Player   `json:"player"`
	Mission          *Mission `json:"mission"`
}
