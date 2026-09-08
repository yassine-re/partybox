ALTER TABLE games DROP CONSTRAINT games_mode_check;
ALTER TABLE games ADD CONSTRAINT games_mode_check
    CHECK (mode IN ('secret_missions', 'treasure_hunt', 'chaos'));

ALTER TABLE missions DROP CONSTRAINT missions_mode_check;
ALTER TABLE missions ADD CONSTRAINT missions_mode_check
    CHECK (mode IN ('secret_missions', 'treasure_hunt', 'chaos'));

ALTER TABLE player_missions DROP CONSTRAINT player_missions_status_check;
ALTER TABLE player_missions ADD CONSTRAINT player_missions_status_check
    CHECK (status IN ('assigned', 'completed', 'cancelled'));

CREATE TABLE chaos_states (
    game_id uuid PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE,
    event_type text CHECK (event_type IN ('double_trouble', 'bounty', 'mission_shuffle')),
    target_player_id uuid REFERENCES players(id) ON DELETE SET NULL,
    remaining_uses integer NOT NULL DEFAULT 0 CHECK (remaining_uses >= 0),
    completions_since_event integer NOT NULL DEFAULT 0 CHECK (completions_since_event >= 0),
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    event_sequence integer NOT NULL DEFAULT 0 CHECK (event_sequence >= 0),
    started_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now()
);
