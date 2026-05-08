CREATE TABLE IF NOT EXISTS user_social_links (
    id        TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id   TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform  TEXT NOT NULL,
    url       TEXT NOT NULL DEFAULT '',
    username  TEXT NOT NULL DEFAULT '',
    verified  BOOLEAN NOT NULL DEFAULT false,
    created   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_social_links_user_platform ON user_social_links (user_id, platform);
CREATE INDEX IF NOT EXISTS idx_user_social_links_platform ON user_social_links (platform);
