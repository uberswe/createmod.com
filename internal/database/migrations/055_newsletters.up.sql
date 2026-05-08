CREATE TABLE IF NOT EXISTS newsletter_subscribers (
    id              TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    email           TEXT NOT NULL,
    user_id         TEXT REFERENCES users(id) ON DELETE SET NULL,
    type            TEXT NOT NULL DEFAULT 'trending',
    frequency       TEXT NOT NULL DEFAULT 'weekly',
    confirmed       BOOLEAN NOT NULL DEFAULT false,
    confirm_token   TEXT NOT NULL DEFAULT '',
    unsubscribe_token TEXT NOT NULL DEFAULT '',
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_newsletter_subscribers_email_type ON newsletter_subscribers (email, type);

CREATE TABLE IF NOT EXISTS search_alerts (
    id                TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id           TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    query             TEXT NOT NULL,
    filters           JSONB NOT NULL DEFAULT '{}',
    frequency         TEXT NOT NULL DEFAULT 'weekly',
    last_checked      TIMESTAMPTZ,
    last_notified     TIMESTAMPTZ,
    active            BOOLEAN NOT NULL DEFAULT true,
    unsubscribe_token TEXT NOT NULL DEFAULT '',
    created           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_search_alerts_user ON search_alerts (user_id);

CREATE TABLE IF NOT EXISTS section_subscriptions (
    id                TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id           TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_type TEXT NOT NULL,
    target_id         TEXT NOT NULL DEFAULT '',
    frequency         TEXT NOT NULL DEFAULT 'weekly',
    unsubscribe_token TEXT NOT NULL DEFAULT '',
    created           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_section_subscriptions_pair ON section_subscriptions (user_id, subscription_type, target_id);

CREATE TABLE IF NOT EXISTS newsletter_issues (
    id        TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    type      TEXT NOT NULL DEFAULT '',
    subject   TEXT NOT NULL DEFAULT '',
    html_body TEXT NOT NULL DEFAULT '',
    slug      TEXT NOT NULL UNIQUE,
    sent_at   TIMESTAMPTZ,
    created   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
