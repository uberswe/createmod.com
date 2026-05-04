-- Drop and recreate materialized view with zero_result_count column.
DROP MATERIALIZED VIEW IF EXISTS search_query_counts;

CREATE MATERIALIZED VIEW search_query_counts AS
SELECT
    LEFT(query, 500) AS query,
    COUNT(*) AS search_count,
    COUNT(*) FILTER (WHERE results_count = 0) AS zero_result_count
FROM searches
GROUP BY LEFT(query, 500)
ORDER BY search_count DESC;

CREATE UNIQUE INDEX idx_search_query_counts_query
    ON search_query_counts (query);

-- Table for persisting search result clicks.
CREATE TABLE search_clicks (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    query       TEXT NOT NULL DEFAULT '',
    result_id   TEXT NOT NULL DEFAULT '',
    position    INTEGER NOT NULL DEFAULT 0,
    user_id     TEXT,
    ip_address  TEXT NOT NULL DEFAULT '',
    created     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_search_clicks_created ON search_clicks (created);
CREATE INDEX idx_search_clicks_query ON search_clicks (query);

-- Table for tracking search-to-view conversions (user arrived at a schematic from search).
CREATE TABLE search_conversions (
    id            TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    query         TEXT NOT NULL DEFAULT '',
    schematic_id  TEXT NOT NULL DEFAULT '',
    user_id       TEXT,
    ip_address    TEXT NOT NULL DEFAULT '',
    created       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_search_conversions_created ON search_conversions (created);
CREATE INDEX idx_search_conversions_query ON search_conversions (query);
