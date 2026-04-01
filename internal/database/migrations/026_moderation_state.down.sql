-- Reverse: re-add boolean columns from moderation_state.
ALTER TABLE schematics ADD COLUMN moderated BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE schematics ADD COLUMN blacklisted BOOLEAN NOT NULL DEFAULT false;

-- Backfill booleans from state.
UPDATE schematics SET moderated = true  WHERE moderation_state IN ('published', 'approved');
UPDATE schematics SET blacklisted = true WHERE moderation_state IN ('rejected', 'flagged');
-- auto_review keeps defaults (moderated=false, blacklisted=false).
-- deleted rows keep defaults (visibility controlled by deleted column).

-- Drop state column and index.
DROP INDEX IF EXISTS idx_schematics_moderation_state;
ALTER TABLE schematics DROP COLUMN moderation_state;

-- Re-create old index.
CREATE INDEX IF NOT EXISTS idx_schematics_moderated ON schematics (moderated);
