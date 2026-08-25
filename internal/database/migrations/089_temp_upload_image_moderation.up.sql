-- Moderate API-uploaded temp images before they are served or displayed, so
-- nudity/other disallowed content can't be hosted via the upload API. Violence
-- is allowed (Minecraft weapon/vehicle builds). Status: pending (just uploaded,
-- awaiting the async check) | approved (served + shown) | rejected (S3 object
-- deleted, never served). (#1646)
ALTER TABLE temp_upload_images
    ADD COLUMN IF NOT EXISTS moderation_status TEXT NOT NULL DEFAULT 'pending';

CREATE INDEX IF NOT EXISTS idx_temp_upload_images_token_filename
    ON temp_upload_images (token, filename);
