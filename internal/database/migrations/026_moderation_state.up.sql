-- Add moderation_state column with a proper state machine replacing boolean moderated/blacklisted.
ALTER TABLE schematics ADD COLUMN moderation_state TEXT NOT NULL DEFAULT 'auto_review';

-- Backfill existing rows from the current boolean columns.
-- Order matters: check deleted first, then blacklisted, then moderated.
UPDATE schematics SET moderation_state = 'deleted'   WHERE deleted IS NOT NULL;
UPDATE schematics SET moderation_state = 'rejected'  WHERE blacklisted = true AND deleted IS NULL;
UPDATE schematics SET moderation_state = 'published'  WHERE moderated = true AND blacklisted = false AND deleted IS NULL;
-- Remaining rows (moderated = false AND blacklisted = false AND deleted IS NULL) keep default 'auto_review'.

-- Drop old columns.
ALTER TABLE schematics DROP COLUMN moderated;
ALTER TABLE schematics DROP COLUMN blacklisted;

-- Drop old index if it exists and create new one.
DROP INDEX IF EXISTS idx_schematics_moderated;
CREATE INDEX idx_schematics_moderation_state ON schematics (moderation_state);
