DROP INDEX IF EXISTS idx_temp_upload_images_token_filename;
ALTER TABLE temp_upload_images DROP COLUMN IF EXISTS moderation_status;
