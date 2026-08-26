DROP INDEX IF EXISTS idx_schematics_human_review_requested;
ALTER TABLE schematics DROP COLUMN IF EXISTS human_review_requested;
ALTER TABLE schematics DROP COLUMN IF EXISTS moderation_reviewed_by;
