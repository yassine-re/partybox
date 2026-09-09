-- Removes verdict history and attempt counts; never changes scores/assignments.
DROP TABLE mission_proofs;
ALTER TABLE player_missions DROP COLUMN proof_attempts;
