ALTER TABLE games
    ADD COLUMN duration_minutes integer NOT NULL DEFAULT 30 CHECK (duration_minutes IN (0,15,30,60)),
    ADD COLUMN ends_at timestamptz,
    ADD COLUMN finished_at timestamptz,
    ADD COLUMN leaderboard_visibility text NOT NULL DEFAULT 'visible' CHECK (leaderboard_visibility IN ('visible','hidden')),
    ADD COLUMN validation_mode text NOT NULL DEFAULT 'trust' CHECK (validation_mode IN ('trust','peer'));

-- Existing running games keep their previous unbounded behavior.
UPDATE games SET duration_minutes=0 WHERE status='playing';
UPDATE games SET finished_at=ended_at WHERE status='ended';
CREATE INDEX games_due_idx ON games(ends_at) WHERE status='playing' AND ends_at IS NOT NULL;

CREATE TABLE mission_validations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id uuid NOT NULL UNIQUE REFERENCES player_missions(id) ON DELETE CASCADE,
    game_id uuid NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    player_id uuid NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected','expired')),
    validator_id uuid REFERENCES players(id),
    requested_at timestamptz NOT NULL DEFAULT now(),
    resolved_at timestamptz
);
CREATE INDEX mission_validations_pending_idx ON mission_validations(game_id,requested_at) WHERE status='pending';
ALTER TABLE player_missions ADD COLUMN awarded_points integer NOT NULL DEFAULT 0 CHECK (awarded_points >= 0);
