-- Existing games and missions keep their Secret Missions identity.
ALTER TABLE games DROP CONSTRAINT games_mode_check;
ALTER TABLE games ADD CONSTRAINT games_mode_check
    CHECK (mode IN ('secret_missions', 'treasure_hunt'));

ALTER TABLE missions ADD COLUMN mode text NOT NULL DEFAULT 'secret_missions';
ALTER TABLE missions ADD CONSTRAINT missions_mode_check
    CHECK (mode IN ('secret_missions', 'treasure_hunt'));

-- No index on this 33-row catalog: a sequential scan is cheaper and simpler.
