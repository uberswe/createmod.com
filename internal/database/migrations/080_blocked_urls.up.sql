-- Admin-managed blocked URLs. Requests whose path + query exactly match an
-- entry are served a 404, e.g. to honor DMCA takedown notices without
-- removing unrelated content.
CREATE TABLE IF NOT EXISTS blocked_urls (
    id         TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    url        TEXT NOT NULL,
    note       TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_blocked_urls_url ON blocked_urls (url);
