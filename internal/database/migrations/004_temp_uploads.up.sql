-- Extend temp_uploads with fields needed for direct S3 storage (replaces PocketBase).
-- Add unique constraint on token
ALTER TABLE temp_uploads ADD CONSTRAINT temp_uploads_token_unique UNIQUE (token);

-- Add missing columns for direct S3 storage
ALTER TABLE temp_uploads ADD COLUMN IF NOT EXISTS uploaded_by TEXT NOT NULL DEFAULT '';
ALTER TABLE temp_uploads ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
ALTER TABLE temp_uploads ADD COLUMN IF NOT EXISTS minecraft_version TEXT NOT NULL DEFAULT '';
ALTER TABLE temp_uploads ADD COLUMN IF NOT EXISTS createmod_version TEXT NOT NULL DEFAULT '';
ALTER TABLE temp_uploads ADD COLUMN IF NOT EXISTS nbt_s3_key TEXT NOT NULL DEFAULT '';
ALTER TABLE temp_uploads ADD COLUMN IF NOT EXISTS image_s3_key TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_temp_uploads_checksum ON temp_uploads(checksum);

-- Rebuild temp_upload_files to reference token instead of temp_upload_id,
-- and add all the NBT parsing columns needed for the upload flow.
DROP TABLE IF EXISTS temp_upload_files;
CREATE TABLE temp_upload_files (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    token TEXT NOT NULL REFERENCES temp_uploads(token) ON DELETE CASCADE,
    filename TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    size BIGINT NOT NULL DEFAULT 0,
    checksum TEXT NOT NULL DEFAULT '',
    block_count INTEGER NOT NULL DEFAULT 0,
    dim_x INTEGER NOT NULL DEFAULT 0,
    dim_y INTEGER NOT NULL DEFAULT 0,
    dim_z INTEGER NOT NULL DEFAULT 0,
    mods JSONB DEFAULT '[]',
    materials JSONB DEFAULT '[]',
    nbt_s3_key TEXT NOT NULL DEFAULT '',
    created TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_temp_upload_files_token ON temp_upload_files(token);
