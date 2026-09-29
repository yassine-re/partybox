ALTER TABLE games DROP CONSTRAINT games_duration_minutes_check;
ALTER TABLE games ADD CONSTRAINT games_duration_minutes_check
    CHECK (duration_minutes IN (0, 5, 15, 30, 60));
