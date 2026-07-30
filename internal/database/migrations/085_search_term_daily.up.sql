-- Per-day, per-term search counts, pre-aggregated from the searches table by a
-- daily background job. The site-stats page (Daily Search Volume, Trending
-- Search Terms, Top Searches) reads from here instead of scanning millions of
-- raw search rows on every load. Only complete days are stored — the current
-- partial day is never aggregated, so charts don't show a misleading dip.
CREATE TABLE IF NOT EXISTS search_term_daily (
    day          DATE   NOT NULL,
    query        TEXT   NOT NULL,
    search_count BIGINT NOT NULL,
    PRIMARY KEY (day, query)
);

-- Range/window scans filter by day.
CREATE INDEX IF NOT EXISTS idx_search_term_daily_day ON search_term_daily (day);
