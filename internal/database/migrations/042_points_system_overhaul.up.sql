-- Drop the unique constraint that prevents repeat point awards.
-- The old constraint only allows one point_log entry per (user_id, reason).
-- The new rules award points per milestone (e.g. every 10K views per schematic),
-- so we need (user_id, reason, reference_id) uniqueness instead.
DROP INDEX IF EXISTS idx_point_log_user_reason;

ALTER TABLE point_log ADD COLUMN IF NOT EXISTS reference_id TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_point_log_user_reason_ref ON point_log (user_id, reason, reference_id);
CREATE INDEX IF NOT EXISTS idx_point_log_user_reason ON point_log (user_id, reason);
