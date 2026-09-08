package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"partybox/backend/internal/models"
)

type Repository struct{ Pool *pgxpool.Pool }

type querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func normalize(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%w : partie déjà active ou pseudo déjà utilisé", models.ErrConflict)
	}
	if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return models.ErrNotFound
	}
	return err
}

type Transaction struct{ tx pgx.Tx }

func (r *Repository) Transaction(ctx context.Context, fn func(*Transaction) error) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = fn(&Transaction{tx: tx}); err != nil {
		return normalize(err)
	}
	return normalize(tx.Commit(ctx))
}

const gameColumns = `id, box_id, name, mode, status, created_at, started_at, ended_at`

func game(ctx context.Context, q querier, id string, lock bool) (models.Game, error) {
	var g models.Game
	sql := `SELECT ` + gameColumns + ` FROM games WHERE id=$1`
	if lock {
		sql += ` FOR UPDATE`
	}
	err := q.QueryRow(ctx, sql, id).Scan(&g.ID, &g.BoxID, &g.Name, &g.Mode, &g.Status, &g.CreatedAt, &g.StartedAt, &g.EndedAt)
	return g, normalize(err)
}

func player(ctx context.Context, q querier, id string) (models.Player, error) {
	var p models.Player
	err := q.QueryRow(ctx, `SELECT id, game_id, name, score, is_host,
		(SELECT count(*) FROM player_missions WHERE player_id=p.id AND status='completed')
		FROM players p WHERE id=$1`, id).Scan(&p.ID, &p.GameID, &p.Name, &p.Score, &p.IsHost, &p.CompletedMissions)
	return p, normalize(err)
}

func players(ctx context.Context, q querier, gameID string) ([]models.Player, error) {
	rows, err := q.Query(ctx, `SELECT id, game_id, name, score, is_host,
		(SELECT count(*) FROM player_missions WHERE player_id=p.id AND status='completed')
		FROM players p WHERE game_id=$1 ORDER BY score DESC, created_at, id`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.Player{}
	for rows.Next() {
		var p models.Player
		if err = rows.Scan(&p.ID, &p.GameID, &p.Name, &p.Score, &p.IsHost, &p.CompletedMissions); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *Repository) Box(ctx context.Context, id string) (models.Box, error) {
	var b models.Box
	err := r.Pool.QueryRow(ctx, `SELECT id, name FROM boxes WHERE id=$1`, id).Scan(&b.ID, &b.Name)
	if err != nil {
		return b, normalize(err)
	}
	var g models.Game
	err = r.Pool.QueryRow(ctx, `SELECT `+gameColumns+` FROM games WHERE box_id=$1 AND status IN ('lobby','playing')`, id).
		Scan(&g.ID, &g.BoxID, &g.Name, &g.Mode, &g.Status, &g.CreatedAt, &g.StartedAt, &g.EndedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return b, nil
	}
	if err != nil {
		return b, err
	}
	b.ActiveGame = &g
	return b, nil
}

func (r *Repository) Game(ctx context.Context, id string) (models.Game, error) {
	// One repeatable-read snapshot prevents a mixed status/player list during Start.
	tx, err := r.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return models.Game{}, err
	}
	defer tx.Rollback(ctx)
	g, err := game(ctx, tx, id, false)
	if err != nil {
		return g, err
	}
	g.Players, err = players(ctx, tx, id)
	if err != nil {
		return g, err
	}
	return g, tx.Commit(ctx)
}

func (r *Repository) Authenticate(ctx context.Context, hash string) (models.Player, error) {
	var id string
	err := r.Pool.QueryRow(ctx, `SELECT id FROM players WHERE token_hash=$1`, hash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Player{}, models.ErrUnauthorized
	}
	if err != nil {
		return models.Player{}, err
	}
	return player(ctx, r.Pool, id)
}

func insertPlayer(ctx context.Context, q querier, gameID, name, hash string, host bool) (models.Player, error) {
	var id string
	err := q.QueryRow(ctx, `INSERT INTO players(id,game_id,name,token_hash,is_host) VALUES(gen_random_uuid(),$1,$2,$3,$4) RETURNING id`, gameID, name, hash, host).Scan(&id)
	if err != nil {
		return models.Player{}, err
	}
	return player(ctx, q, id)
}

// Prefer unseen missions, then the least recently played; never repeat immediately
// when the mode's catalog contains another choice. Random breaks ties in this small catalog.
func assign(ctx context.Context, q querier, playerID string) error {
	var id int
	err := q.QueryRow(ctx, `SELECT m.id FROM missions m
		LEFT JOIN player_missions pm ON pm.mission_id=m.id AND pm.player_id=$1
		WHERE m.mode = (SELECT g.mode FROM players p JOIN games g ON g.id=p.game_id WHERE p.id=$1)
		GROUP BY m.id ORDER BY max(pm.assigned_at) ASC NULLS FIRST, random() LIMIT 1`, playerID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w : aucune mission disponible, lancer le seed", models.ErrConflict)
	}
	if err != nil {
		return err
	}
	_, err = q.Exec(ctx, `INSERT INTO player_missions(id,player_id,mission_id) VALUES(gen_random_uuid(),$1,$2)`, playerID, id)
	return err
}

func currentMission(ctx context.Context, q querier, playerID string) (*models.Mission, error) {
	var m models.Mission
	err := q.QueryRow(ctx, `SELECT pm.id,m.id,m.text,m.points,m.category,m.difficulty
		FROM player_missions pm JOIN missions m ON m.id=pm.mission_id
		WHERE pm.player_id=$1 AND pm.status='assigned'`, playerID).Scan(&m.ID, &m.MissionID, &m.Text, &m.Points, &m.Category, &m.Difficulty)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &m, err
}

func (r *Repository) Mission(ctx context.Context, p models.Player) (*models.Mission, error) {
	return currentMission(ctx, r.Pool, p.ID)
}
