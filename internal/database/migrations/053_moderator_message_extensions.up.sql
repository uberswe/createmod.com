ALTER TABLE moderation_threads ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ;
ALTER TABLE moderation_threads ADD COLUMN IF NOT EXISTS resolved_by TEXT DEFAULT '';
ALTER TABLE moderation_threads ADD COLUMN IF NOT EXISTS creator_notified BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE moderation_threads ADD COLUMN IF NOT EXISTS last_message_at TIMESTAMPTZ;
ALTER TABLE moderation_threads ADD COLUMN IF NOT EXISTS creator_last_read TIMESTAMPTZ;
