CREATE TABLE IF NOT EXISTS download_tokens (
    id         TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    token      TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used       BOOLEAN NOT NULL DEFAULT false,
    created    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_download_tokens_token ON download_tokens (token) WHERE used = false;
