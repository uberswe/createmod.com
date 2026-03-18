-- Clean existing search entries with control characters (bot traffic)
DELETE FROM searches WHERE query ~ E'[\\x00-\\x1F\\x7F]';

-- Clean search entries with excessively long queries (bot traffic)
DELETE FROM searches WHERE LENGTH(query) > 500;

-- Materialized view for top search queries (replaces slow GROUP BY on searches table)
CREATE MATERIALIZED VIEW IF NOT EXISTS search_query_counts AS
SELECT LEFT(query, 500) AS query, COUNT(*) AS search_count
FROM searches
GROUP BY LEFT(query, 500)
ORDER BY search_count DESC;

CREATE UNIQUE INDEX IF NOT EXISTS idx_search_query_counts_query
    ON search_query_counts (query);
