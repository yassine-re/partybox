-- The pre-004 schema cannot represent game-scoped catalogs. Remove their
-- assignment history first so the missions FK does not make rollback fail.
DELETE FROM player_missions
WHERE mission_id IN (SELECT id FROM missions WHERE source = 'ai');
DELETE FROM missions WHERE source = 'ai';
ALTER TABLE missions DROP CONSTRAINT IF EXISTS missions_source_game_id_check;
ALTER TABLE missions DROP CONSTRAINT IF EXISTS missions_source_check;
DROP INDEX IF EXISTS missions_game_id_idx;
ALTER TABLE missions DROP COLUMN IF EXISTS game_id;
ALTER TABLE missions DROP COLUMN IF EXISTS source;
