-- Run with psql --single-transaction --set ON_ERROR_STOP=1.
-- Refuse to silently destroy Chaos games, assignments or their history.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM games WHERE mode = 'chaos')
       OR EXISTS (
           SELECT 1 FROM player_missions pm JOIN missions m ON m.id = pm.mission_id
           WHERE m.mode = 'chaos' OR pm.status = 'cancelled'
       ) THEN
        RAISE EXCEPTION 'Rollback blocked: Chaos games or assignments still exist. Archive them explicitly first.';
    END IF;
END $$;

DROP TABLE chaos_states;
DELETE FROM missions WHERE mode = 'chaos';

ALTER TABLE player_missions DROP CONSTRAINT player_missions_status_check;
ALTER TABLE player_missions ADD CONSTRAINT player_missions_status_check
    CHECK (status IN ('assigned', 'completed'));

ALTER TABLE missions DROP CONSTRAINT missions_mode_check;
ALTER TABLE missions ADD CONSTRAINT missions_mode_check
    CHECK (mode IN ('secret_missions', 'treasure_hunt'));

ALTER TABLE games DROP CONSTRAINT games_mode_check;
ALTER TABLE games ADD CONSTRAINT games_mode_check
    CHECK (mode IN ('secret_missions', 'treasure_hunt'));
