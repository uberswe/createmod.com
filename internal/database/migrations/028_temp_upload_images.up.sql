CREATE TABLE temp_upload_images (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    token TEXT NOT NULL REFERENCES temp_uploads(token) ON DELETE CASCADE,
    filename TEXT NOT NULL DEFAULT '',
    size BIGINT NOT NULL DEFAULT 0,
    s3_key TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_temp_upload_images_token ON temp_upload_images(token);
