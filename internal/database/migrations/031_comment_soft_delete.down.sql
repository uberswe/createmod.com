DROP INDEX IF EXISTS idx_comments_deleted;
ALTER TABLE comments DROP COLUMN IF EXISTS deleted;
