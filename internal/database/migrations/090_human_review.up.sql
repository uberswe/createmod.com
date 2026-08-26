-- Track whether a schematic's current moderation outcome came from the automated
-- system or a human moderator, and let authors ask for a human to look when the
-- automated review may have been wrong. (#1646)
ALTER TABLE schematics ADD COLUMN IF NOT EXISTS moderation_reviewed_by TEXT NOT NULL DEFAULT '';
ALTER TABLE schematics ADD COLUMN IF NOT EXISTS human_review_requested BOOLEAN NOT NULL DEFAULT false;

-- Backfill the knowable cases: automated policy flags and (auto) quality limits
-- are 'system'; moderator-only outcomes are 'human'.
UPDATE schematics SET moderation_reviewed_by = 'system'
  WHERE moderation_reviewed_by = '' AND moderation_state IN ('flagged', 'published_limited');
UPDATE schematics SET moderation_reviewed_by = 'human'
  WHERE moderation_reviewed_by = '' AND moderation_state IN ('changes_requested', 'rejected', 'rejected_fixable', 'rejected_final');

-- Surface author-requested human reviews in the moderator queue.
CREATE INDEX IF NOT EXISTS idx_schematics_human_review_requested
  ON schematics (human_review_requested) WHERE human_review_requested = true AND deleted IS NULL;
