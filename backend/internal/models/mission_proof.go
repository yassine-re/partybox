package models

import "time"

type MissionProof struct {
	ID           string    `json:"id"`
	AssignmentID string    `json:"assignment_id"`
	Verdict      string    `json:"verdict"`
	Confidence   float64   `json:"confidence"`
	Reason       string    `json:"reason"`
	CreatedAt    time.Time `json:"created_at"`
}
