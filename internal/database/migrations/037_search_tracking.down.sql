DROP TABLE IF EXISTS search_conversions;
DROP TABLE IF EXISTS search_clicks;

DROP MATERIALIZED VIEW IF EXISTS search_query_counts;

CREATE MATERIALIZED VIEW IF NOT EXISTS search_query_counts AS
SELECT LEFT(query, 500) AS query, COUNT(*) AS search_count
FROM searches
GROUP BY LEFT(query, 500)
ORDER BY search_count DESC;

CREATE UNIQUE INDEX IF NOT EXISTS idx_search_query_counts_query
    ON search_query_counts (query);
