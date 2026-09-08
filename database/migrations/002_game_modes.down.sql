-- Run with psql --single-transaction --set ON_ERROR_STOP=1.
-- Refuse to silently destroy or relabel Treasure Hunt history.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM games WHERE mode = 'treasure_hunt')
       OR EXISTS (
           SELECT 1 FROM player_missions pm JOIN missions m ON m.id = pm.mission_id
           WHERE m.mode = 'treasure_hunt'
       ) THEN
        RAISE EXCEPTION 'Rollback blocked: Treasure Hunt games or assignments still exist. Archive them explicitly first.';
    END IF;
END $$;

DELETE FROM missions WHERE mode = 'treasure_hunt';
ALTER TABLE missions DROP COLUMN mode;
ALTER TABLE games DROP CONSTRAINT games_mode_check;
ALTER TABLE games ADD CONSTRAINT games_mode_check CHECK (mode = 'secret_missions');
