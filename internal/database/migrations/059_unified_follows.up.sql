-- Unify user_follows and section_subscriptions into a single follows table
-- that supports following users, categories, feeds (latest/trending), searches, and mods.

-- 1. Create the new unified table
CREATE TABLE unified_follows (
    id               TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    user_id          TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    follow_type      TEXT NOT NULL,    -- 'user','category','latest','trending','highest_rated','search','mod'
    target_id        TEXT NOT NULL DEFAULT '',
    email_frequency  TEXT NOT NULL DEFAULT 'off', -- 'realtime','daily','weekly','off'
    unsubscribe_token TEXT NOT NULL DEFAULT '',
    last_notified    TIMESTAMPTZ,
    created          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_unified_follows_unique ON unified_follows (user_id, follow_type, target_id);
CREATE INDEX idx_unified_follows_user ON unified_follows (user_id);
CREATE INDEX idx_unified_follows_target ON unified_follows (follow_type, target_id);
CREATE INDEX idx_unified_follows_email ON unified_follows (email_frequency) WHERE email_frequency != 'off';

-- 2. Migrate existing user follows
INSERT INTO unified_follows (id, user_id, follow_type, target_id, email_frequency, created)
SELECT id, follower_id, 'user', followed_id, 'off', created
FROM user_follows
ON CONFLICT DO NOTHING;

-- 3. Migrate section subscriptions (category/tag follows with email)
INSERT INTO unified_follows (user_id, follow_type, target_id, email_frequency, unsubscribe_token, created)
SELECT user_id, subscription_type, target_id, frequency, unsubscribe_token, created
FROM section_subscriptions
WHERE frequency != 'off'
ON CONFLICT (user_id, follow_type, target_id) DO UPDATE SET
    email_frequency = EXCLUDED.email_frequency,
    unsubscribe_token = EXCLUDED.unsubscribe_token;

-- 4. Drop old tables
DROP TABLE user_follows;
DROP TABLE section_subscriptions;

-- 5. Rename to user_follows
ALTER TABLE unified_follows RENAME TO user_follows;
ALTER INDEX idx_unified_follows_unique RENAME TO idx_user_follows_unique;
ALTER INDEX idx_unified_follows_user RENAME TO idx_user_follows_user;
ALTER INDEX idx_unified_follows_target RENAME TO idx_user_follows_target;
ALTER INDEX idx_unified_follows_email RENAME TO idx_user_follows_email;
