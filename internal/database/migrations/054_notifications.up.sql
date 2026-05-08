CREATE TABLE IF NOT EXISTS notifications (
    id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type         TEXT NOT NULL,
    title        TEXT NOT NULL DEFAULT '',
    body         TEXT NOT NULL DEFAULT '',
    url          TEXT NOT NULL DEFAULT '',
    actor_id     TEXT DEFAULT '',
    reference_id TEXT DEFAULT '',
    read         BOOLEAN NOT NULL DEFAULT false,
    created      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notifications_user_read ON notifications (user_id, read, created DESC);

CREATE TABLE IF NOT EXISTS notification_preferences (
    id       TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id  TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category TEXT NOT NULL,
    email    TEXT NOT NULL DEFAULT 'off',
    web      BOOLEAN NOT NULL DEFAULT true,
    created  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_notification_preferences_pair ON notification_preferences (user_id, category);
