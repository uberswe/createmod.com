-- Admin-managed shared HMAC secrets for the mod / partner API. The value is
-- stored in plaintext because HMAC verification needs the raw secret (the
-- values are treated as public). Env-var secrets (MOD_DOWNLOAD_SECRET) remain
-- accepted in addition to these.
CREATE TABLE IF NOT EXISTS mod_secrets (
    id         TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    label      TEXT NOT NULL DEFAULT '',
    note       TEXT NOT NULL DEFAULT '',
    secret     TEXT NOT NULL,
    active     BOOLEAN NOT NULL DEFAULT true,
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Hot path lists only active secrets.
CREATE INDEX IF NOT EXISTS idx_mod_secrets_active ON mod_secrets (active) WHERE active;
