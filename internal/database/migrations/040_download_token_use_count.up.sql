ALTER TABLE download_tokens ADD COLUMN IF NOT EXISTS use_count INTEGER NOT NULL DEFAULT 0;
UPDATE download_tokens SET use_count = CASE WHEN used THEN 1 ELSE 0 END;
DROP INDEX IF EXISTS idx_download_tokens_token;
CREATE INDEX idx_download_tokens_token ON download_tokens (token) WHERE use_count < 5;
