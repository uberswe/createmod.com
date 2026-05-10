-- Recreate old user_follows table
CREATE TABLE old_user_follows (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    follower_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    followed_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_old_user_follows_unique ON old_user_follows (follower_id, followed_id);
CREATE INDEX idx_old_user_follows_followed ON old_user_follows (followed_id);
CREATE INDEX idx_old_user_follows_follower ON old_user_follows (follower_id);

INSERT INTO old_user_follows (id, follower_id, followed_id, created)
SELECT id, user_id, target_id, created
FROM user_follows
WHERE follow_type = 'user'
ON CONFLICT DO NOTHING;

-- Recreate section_subscriptions
CREATE TABLE section_subscriptions (
    id                TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    user_id           TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_type TEXT NOT NULL,
    target_id         TEXT NOT NULL DEFAULT '',
    frequency         TEXT NOT NULL DEFAULT 'weekly',
    unsubscribe_token TEXT NOT NULL DEFAULT '',
    created           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_section_subs_unique ON section_subscriptions (user_id, subscription_type, target_id);

INSERT INTO section_subscriptions (user_id, subscription_type, target_id, frequency, unsubscribe_token, created)
SELECT user_id, follow_type, target_id, email_frequency, unsubscribe_token, created
FROM user_follows
WHERE follow_type IN ('category', 'tag') AND email_frequency != 'off'
ON CONFLICT DO NOTHING;

-- Drop unified and rename
DROP TABLE user_follows;
ALTER TABLE old_user_follows RENAME TO user_follows;
ALTER INDEX idx_old_user_follows_unique RENAME TO idx_user_follows_unique;
ALTER INDEX idx_old_user_follows_followed RENAME TO idx_user_follows_followed;
ALTER INDEX idx_old_user_follows_follower RENAME TO idx_user_follows_follower;
