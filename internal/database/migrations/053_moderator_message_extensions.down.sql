ALTER TABLE moderation_threads DROP COLUMN IF EXISTS creator_last_read;
ALTER TABLE moderation_threads DROP COLUMN IF EXISTS last_message_at;
ALTER TABLE moderation_threads DROP COLUMN IF EXISTS creator_notified;
ALTER TABLE moderation_threads DROP COLUMN IF EXISTS resolved_by;
ALTER TABLE moderation_threads DROP COLUMN IF EXISTS resolved_at;
