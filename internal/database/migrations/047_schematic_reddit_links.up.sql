CREATE TABLE IF NOT EXISTS schematic_reddit_links (
    id              TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    schematic_id    TEXT NOT NULL REFERENCES schematics(id) ON DELETE CASCADE,
    reddit_url      TEXT NOT NULL UNIQUE,
    subreddit       TEXT NOT NULL DEFAULT '',
    post_title      TEXT NOT NULL DEFAULT '',
    upvotes         INTEGER NOT NULL DEFAULT 0,
    comment_count   INTEGER NOT NULL DEFAULT 0,
    thumbnail_url   TEXT NOT NULL DEFAULT '',
    last_fetched    TIMESTAMPTZ,
    created         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_schematic_reddit_links_schematic ON schematic_reddit_links (schematic_id);
