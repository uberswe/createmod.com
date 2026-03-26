-- Prune single-use searches older than 90 days to reduce searches table bloat
-- and materialized view refresh cost. These one-off queries have no analytics value.
WITH single_use AS (
  SELECT LEFT(query, 500) AS q
  FROM searches
  GROUP BY LEFT(query, 500)
  HAVING COUNT(*) = 1
)
DELETE FROM searches s
USING single_use su
WHERE LEFT(s.query, 500) = su.q
  AND s.created < NOW() - INTERVAL '90 days';
