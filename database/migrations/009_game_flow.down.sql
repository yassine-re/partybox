ALTER TABLE player_missions DROP COLUMN awarded_points;
DROP TABLE mission_validations;
DROP INDEX games_due_idx;
ALTER TABLE games DROP COLUMN duration_minutes, DROP COLUMN ends_at, DROP COLUMN finished_at,
    DROP COLUMN leaderboard_visibility, DROP COLUMN validation_mode;
