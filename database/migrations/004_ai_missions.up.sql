-- Distinguish seed missions from game-specific AI-generated missions.
ALTER TABLE missions ADD COLUMN source text NOT NULL DEFAULT 'seed';
ALTER TABLE missions ADD CONSTRAINT missions_source_check CHECK (source IN ('seed', 'ai'));

ALTER TABLE missions ADD COLUMN game_id uuid NULL REFERENCES games(id) ON DELETE CASCADE;
ALTER TABLE missions ADD CONSTRAINT missions_source_game_id_check
    CHECK ((source = 'seed' AND game_id IS NULL) OR (source = 'ai' AND game_id IS NOT NULL));

CREATE INDEX missions_game_id_idx ON missions(game_id) WHERE game_id IS NOT NULL;
