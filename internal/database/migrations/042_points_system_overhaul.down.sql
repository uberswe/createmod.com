DROP INDEX IF EXISTS idx_point_log_user_reason;
DROP INDEX IF EXISTS idx_point_log_user_reason_ref;

ALTER TABLE point_log DROP COLUMN IF EXISTS reference_id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_point_log_user_reason ON point_log (user_id, reason);
