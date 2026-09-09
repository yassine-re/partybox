package repositories

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// SetMissionFeedback stores the final rating for an assignment. It reports
// whether the authoritative value changed so retries do not duplicate events.
func (t *Transaction) SetMissionFeedback(ctx context.Context, assignmentID string, rating int) (bool, error) {
	var changed bool
	err := t.tx.QueryRow(ctx, `INSERT INTO mission_feedback(id,assignment_id,rating)
		VALUES(gen_random_uuid(),$1,$2)
		ON CONFLICT (assignment_id) DO UPDATE SET rating=EXCLUDED.rating
		WHERE mission_feedback.rating IS DISTINCT FROM EXCLUDED.rating
		RETURNING true`, assignmentID, rating).Scan(&changed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return changed, err
}
