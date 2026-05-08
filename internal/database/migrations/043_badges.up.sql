CREATE TABLE IF NOT EXISTS badges (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    key         TEXT NOT NULL,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    icon        TEXT NOT NULL DEFAULT '',
    category    TEXT NOT NULL DEFAULT 'achievement',
    threshold   INTEGER NOT NULL DEFAULT 0,
    multi_earn  BOOLEAN NOT NULL DEFAULT false,
    created     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_badges_key ON badges (key);

CREATE TABLE IF NOT EXISTS user_badges (
    id        TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id   TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_id  TEXT NOT NULL REFERENCES badges(id) ON DELETE CASCADE,
    count     INTEGER NOT NULL DEFAULT 1,
    created   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_badges_user_badge ON user_badges (user_id, badge_id);
CREATE INDEX IF NOT EXISTS idx_user_badges_user ON user_badges (user_id);

CREATE TABLE IF NOT EXISTS user_displayed_badges (
    id        TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id   TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_id  TEXT NOT NULL REFERENCES badges(id) ON DELETE CASCADE,
    position  SMALLINT NOT NULL DEFAULT 0,
    created   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_displayed_badges_user_pos ON user_displayed_badges (user_id, position);
CREATE INDEX IF NOT EXISTS idx_user_displayed_badges_user ON user_displayed_badges (user_id);

INSERT INTO badges (key, title, description, icon, category, threshold, multi_earn) VALUES
    ('views_10k',      '10K Views',           'A schematic reached 10,000 views',          'eye',     'achievement', 10000,   true),
    ('views_100k',     '100K Views',          'A schematic reached 100,000 views',         'eye',     'achievement', 100000,  true),
    ('views_1m',       '1M Views',            'A schematic reached 1,000,000 views',       'eye',     'achievement', 1000000, true),
    ('ratings_100',    'Rating Expert',       'Rated 100 schematics',                      'star',    'achievement', 100,     false),
    ('comments_100',   'Commentator',         'Posted 100 comments',                       'message', 'achievement', 100,     false),
    ('patreon_creator','Creator Supporter',   'Supports a creator on Patreon',             'heart',   'supporter',   0,       false),
    ('patreon_mod',    'Mod Supporter',       'Supports the Create mod on Patreon',        'heart',   'supporter',   0,       false),
    ('verified',       'Verified',            'Verified by site administrators',           'check',   'verified',    0,       false)
ON CONFLICT DO NOTHING;
