DROP INDEX IF EXISTS idx_download_tokens_token;
CREATE INDEX idx_download_tokens_token ON download_tokens (token) WHERE used = false;
ALTER TABLE download_tokens DROP COLUMN IF EXISTS use_count;
