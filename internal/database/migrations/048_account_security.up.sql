CREATE TABLE IF NOT EXISTS user_known_ips (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip_address  TEXT NOT NULL,
    user_agent  TEXT NOT NULL DEFAULT '',
    verified    BOOLEAN NOT NULL DEFAULT false,
    last_seen   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_known_ips_pair ON user_known_ips (user_id, ip_address);

CREATE TABLE IF NOT EXISTS ip_verification_codes (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip_address  TEXT NOT NULL,
    code_hash   TEXT NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    used        BOOLEAN NOT NULL DEFAULT false,
    created     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ip_verification_codes_user ON ip_verification_codes (user_id);

CREATE TABLE IF NOT EXISTS user_totp (
    id               TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id          TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    secret_encrypted TEXT NOT NULL,
    enabled          BOOLEAN NOT NULL DEFAULT false,
    verified         BOOLEAN NOT NULL DEFAULT false,
    created          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_totp_backup_codes (
    id        TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id   TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,
    used      BOOLEAN NOT NULL DEFAULT false,
    created   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_user_totp_backup_codes_user ON user_totp_backup_codes (user_id);

CREATE TABLE IF NOT EXISTS user_passkeys (
    id               TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id          TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id    BYTEA NOT NULL UNIQUE,
    public_key       BYTEA NOT NULL,
    attestation_type TEXT NOT NULL DEFAULT '',
    transport        TEXT[] NOT NULL DEFAULT '{}',
    aaguid           BYTEA,
    sign_count       INTEGER NOT NULL DEFAULT 0,
    friendly_name    TEXT NOT NULL DEFAULT '',
    last_used        TIMESTAMPTZ,
    created          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_user_passkeys_user ON user_passkeys (user_id);

CREATE TABLE IF NOT EXISTS user_security_settings (
    id                    TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id               TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    new_ip_verification   BOOLEAN NOT NULL DEFAULT false,
    totp_enabled          BOOLEAN NOT NULL DEFAULT false,
    passkeys_enabled      BOOLEAN NOT NULL DEFAULT false,
    created               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
