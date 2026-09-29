DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM games WHERE duration_minutes = 5) THEN
        RAISE EXCEPTION 'Rollback blocked: five-minute games exist. Archive them explicitly first.';
    END IF;
END $$;

ALTER TABLE games DROP CONSTRAINT games_duration_minutes_check;
ALTER TABLE games ADD CONSTRAINT games_duration_minutes_check
    CHECK (duration_minutes IN (0, 15, 30, 60));
