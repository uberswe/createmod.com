-- Revert temp_upload_files to original schema
DROP TABLE IF EXISTS temp_upload_files;
CREATE TABLE temp_upload_files (
    id              TEXT PRIMARY KEY,
    temp_upload_id  TEXT NOT NULL REFERENCES temp_uploads(id) ON DELETE CASCADE,
    filename        TEXT NOT NULL DEFAULT '',
    original_name   TEXT NOT NULL DEFAULT '',
    size            BIGINT NOT NULL DEFAULT 0,
    mime_type       TEXT NOT NULL DEFAULT '',
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_temp_upload_files_upload ON temp_upload_files (temp_upload_id);

-- Remove added columns from temp_uploads
ALTER TABLE temp_uploads DROP COLUMN IF EXISTS uploaded_by;
ALTER TABLE temp_uploads DROP COLUMN IF EXISTS description;
ALTER TABLE temp_uploads DROP COLUMN IF EXISTS minecraft_version;
ALTER TABLE temp_uploads DROP COLUMN IF EXISTS createmod_version;
ALTER TABLE temp_uploads DROP COLUMN IF EXISTS nbt_s3_key;
ALTER TABLE temp_uploads DROP COLUMN IF EXISTS image_s3_key;
DROP INDEX IF EXISTS idx_temp_uploads_checksum;
ALTER TABLE temp_uploads DROP CONSTRAINT IF EXISTS temp_uploads_token_unique;
