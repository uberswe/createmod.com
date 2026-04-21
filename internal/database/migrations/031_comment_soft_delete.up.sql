-- Add soft-delete column to comments so admins can hide comments without losing data.
ALTER TABLE comments ADD COLUMN deleted TIMESTAMPTZ;

CREATE INDEX idx_comments_deleted ON comments (deleted) WHERE deleted IS NULL;
