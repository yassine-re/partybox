CREATE TABLE game_events (
    id uuid PRIMARY KEY,
    game_id uuid NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    player_id uuid REFERENCES players(id) ON DELETE SET NULL,
    type text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX game_events_game_created_idx ON game_events(game_id, created_at);
CREATE INDEX game_events_type_idx ON game_events(type);
