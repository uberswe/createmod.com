-- Clean existing search entries with control characters (bot traffic)
DELETE FROM searches WHERE query ~ E'[\\x00-\\x1F\\x7F]';

-- Materialized view for top search queries (replaces slow GROUP BY on searches table)
CREATE MATERIALIZED VIEW IF NOT EXISTS search_query_counts AS
SELECT query, COUNT(*) AS search_count
FROM searches
GROUP BY query
ORDER BY search_count DESC;

CREATE UNIQUE INDEX IF NOT EXISTS idx_search_query_counts_query
    ON search_query_counts (query);
