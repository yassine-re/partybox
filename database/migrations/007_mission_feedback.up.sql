CREATE TABLE mission_feedback (
    id uuid PRIMARY KEY,
    assignment_id uuid NOT NULL REFERENCES player_missions(id) ON DELETE CASCADE,
    rating smallint NOT NULL CHECK (rating IN (-1, 0, 1)),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (assignment_id)
);

-- UNIQUE(assignment_id) already provides the index used by feedback lookups.
