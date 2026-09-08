package repositories

import (
	"context"
	"partybox/backend/internal/models"
)

// These operations share the same transaction. The service owns game rules;
// the repository owns SQL and locking. Lock the game before changing its players.
func (t *Transaction) LockBox(ctx context.Context, id string) error {
	var found string
	return t.tx.QueryRow(ctx, `SELECT id FROM boxes WHERE id=$1 FOR UPDATE`, id).Scan(&found)
}

func (t *Transaction) InsertGame(ctx context.Context, boxID, name string, mode models.GameMode) (models.Game, error) {
	var id string
	err := t.tx.QueryRow(ctx, `INSERT INTO games(id,box_id,name,mode) VALUES(gen_random_uuid(),$1,$2,$3) RETURNING id`, boxID, name, mode).Scan(&id)
	if err != nil {
		return models.Game{}, err
	}
	return game(ctx, t.tx, id, false)
}

func (t *Transaction) LockGame(ctx context.Context, id string) (models.Game, error) {
	return game(ctx, t.tx, id, true)
}

func (t *Transaction) InsertPlayer(ctx context.Context, gameID, name, hash string, host bool) (models.Player, error) {
	return insertPlayer(ctx, t.tx, gameID, name, hash, host)
}

func (t *Transaction) Players(ctx context.Context, gameID string) ([]models.Player, error) {
	return players(ctx, t.tx, gameID)
}

func (t *Transaction) Player(ctx context.Context, id string) (models.Player, error) {
	return player(ctx, t.tx, id)
}

func (t *Transaction) AssignMission(ctx context.Context, playerID string) error {
	return assign(ctx, t.tx, playerID)
}

func (t *Transaction) CurrentMission(ctx context.Context, playerID string) (*models.Mission, error) {
	return currentMission(ctx, t.tx, playerID)
}

func (t *Transaction) SetGameStatus(ctx context.Context, gameID, status string) error {
	_, err := t.tx.Exec(ctx, `UPDATE games SET status=$2,
		started_at=CASE WHEN $2='playing' THEN now() ELSE started_at END,
		ended_at=CASE WHEN $2='ended' THEN now() ELSE ended_at END WHERE id=$1`, gameID, status)
	return err
}

func (t *Transaction) Assignment(ctx context.Context, id, playerID string) (status string, points int, err error) {
	err = t.tx.QueryRow(ctx, `SELECT pm.status,m.points FROM player_missions pm JOIN missions m ON m.id=pm.mission_id WHERE pm.id=$1 AND pm.player_id=$2`, id, playerID).Scan(&status, &points)
	return
}

func (t *Transaction) CompleteAssignment(ctx context.Context, id string) error {
	_, err := t.tx.Exec(ctx, `UPDATE player_missions SET status='completed',completed_at=now() WHERE id=$1`, id)
	return err
}

func (t *Transaction) AddScore(ctx context.Context, playerID string, points int) error {
	_, err := t.tx.Exec(ctx, `UPDATE players SET score=score+$1 WHERE id=$2`, points, playerID)
	return err
}

func (t *Transaction) ReplaceAIMissions(ctx context.Context, gameID string, mode models.GameMode, missions []models.MissionInput) error {
	if _, err := t.tx.Exec(ctx, `DELETE FROM missions WHERE game_id=$1 AND source='ai'`, gameID); err != nil {
		return err
	}
	for _, m := range missions {
		if _, err := t.tx.Exec(ctx, `INSERT INTO missions(text,points,category,difficulty,mode,source,game_id)
			VALUES($1,$2,$3,$4,$5,'ai',$6)`, m.Text, m.Points, m.Category, m.Difficulty, mode, gameID); err != nil {
			return err
		}
	}
	return nil
}
