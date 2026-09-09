-- Counts reserved provider calls, including failures, across retries/restarts.
ALTER TABLE player_missions ADD COLUMN proof_attempts integer NOT NULL DEFAULT 0
    CHECK (proof_attempts >= 0);

CREATE TABLE mission_proofs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id uuid NOT NULL REFERENCES player_missions(id) ON DELETE CASCADE,
    verdict text NOT NULL CHECK (verdict IN ('valid', 'invalid', 'uncertain')),
    confidence double precision NOT NULL CHECK (confidence BETWEEN 0 AND 1),
    reason text NOT NULL CHECK (char_length(reason) BETWEEN 1 AND 300),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX mission_proofs_assignment_idx ON mission_proofs(assignment_id, created_at DESC);
