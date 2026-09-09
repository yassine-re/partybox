package repositories

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"partybox/backend/internal/models"
)

type ProofAssignment struct {
	GameID, GameStatus, Status, MissionText string
	Mode                                    models.GameMode
	Attempts                                int
}

func proofAssignment(ctx context.Context, q querier, player models.Player, id string) (ProofAssignment, error) {
	var a ProofAssignment
	err := q.QueryRow(ctx, `SELECT g.id,g.status,g.mode,pm.status,m.text,pm.proof_attempts
		FROM player_missions pm JOIN players p ON p.id=pm.player_id
		JOIN games g ON g.id=p.game_id JOIN missions m ON m.id=pm.mission_id
		WHERE pm.id=$1 AND p.id=$2 AND g.id=$3`, id, player.ID, player.GameID).
		Scan(&a.GameID, &a.GameStatus, &a.Mode, &a.Status, &a.MissionText, &a.Attempts)
	return a, normalize(err)
}

func (r *Repository) ProofAssignment(ctx context.Context, p models.Player, id string) (ProofAssignment, error) {
	return proofAssignment(ctx, r.Pool, p, id)
}

func (t *Transaction) ProofAssignment(ctx context.Context, p models.Player, id string) (ProofAssignment, error) {
	return proofAssignment(ctx, t.tx, p, id)
}

func validProof(ctx context.Context, q querier, id string) (*models.MissionProof, error) {
	var proof models.MissionProof
	err := q.QueryRow(ctx, `SELECT id,assignment_id,verdict,confidence,reason,created_at
		FROM mission_proofs WHERE assignment_id=$1 AND verdict='valid' ORDER BY created_at DESC LIMIT 1`, id).
		Scan(&proof.ID, &proof.AssignmentID, &proof.Verdict, &proof.Confidence, &proof.Reason, &proof.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &proof, err
}

func (r *Repository) ValidProof(ctx context.Context, id string) (*models.MissionProof, error) {
	return validProof(ctx, r.Pool, id)
}

func (t *Transaction) ValidProof(ctx context.Context, id string) (*models.MissionProof, error) {
	return validProof(ctx, t.tx, id)
}

func (t *Transaction) ProofCount(ctx context.Context, id string) (int, error) {
	var count int
	err := t.tx.QueryRow(ctx, `SELECT count(*) FROM mission_proofs WHERE assignment_id=$1`, id).Scan(&count)
	return count, err
}

// Caller holds the game lock. The reservation commits before the network call.
func (t *Transaction) ReserveProofAttempt(ctx context.Context, id string) error {
	_, err := t.tx.Exec(ctx, `UPDATE player_missions SET proof_attempts=proof_attempts+1 WHERE id=$1`, id)
	return err
}

func (t *Transaction) InsertProof(ctx context.Context, proof *models.MissionProof) error {
	return t.tx.QueryRow(ctx, `INSERT INTO mission_proofs(assignment_id,verdict,confidence,reason)
		VALUES($1,$2,$3,$4) RETURNING id,created_at`, proof.AssignmentID, proof.Verdict, proof.Confidence, proof.Reason).
		Scan(&proof.ID, &proof.CreatedAt)
}
