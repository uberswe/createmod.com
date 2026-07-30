-- Queries for the pre-aggregated per-day, per-term search counts
-- (search_term_daily). The write query is run by a daily background job; the
-- read queries back the site-stats page and all exclude the current partial
-- day (day < today, in UTC) so charts show only complete days.

-- name: AggregateSearchTermDaily :exec
-- Aggregate raw searches in [start_ts, end_ts) into per-UTC-day, per-term
-- counts. Idempotent: re-running the same range refreshes the counts.
INSERT INTO search_term_daily (day, query, search_count)
SELECT (created AT TIME ZONE 'UTC')::date AS day,
       LEFT(query, 500) AS query,
       COUNT(*)::BIGINT AS search_count
FROM searches
WHERE created >= @start_ts AND created < @end_ts
GROUP BY 1, 2
ON CONFLICT (day, query) DO UPDATE SET search_count = EXCLUDED.search_count;

-- name: SearchTermDailyDaysBehind :one
-- How many complete UTC days (through yesterday) are not yet aggregated: 0 when
-- yesterday is already stored, N when the last stored day is N days before
-- yesterday, and a large sentinel when the table is empty (first backfill).
SELECT COALESCE(
    ((NOW() AT TIME ZONE 'UTC')::date - 1) - MAX(day),
    9999
)::int AS days_behind
FROM search_term_daily;

-- name: PruneSearchTermDaily :execrows
-- Drop aggregated days older than the raw-search retention window.
DELETE FROM search_term_daily
WHERE day < (NOW() AT TIME ZONE 'UTC')::date - 95;

-- name: DailySearchVolumeAgg :many
-- Total searches per complete day over the last @days days (current day excluded).
SELECT to_char(day, 'YYYY-MM-DD') AS day, SUM(search_count)::BIGINT AS count
FROM search_term_daily
WHERE day >= (NOW() AT TIME ZONE 'UTC')::date - @days::int
  AND day < (NOW() AT TIME ZONE 'UTC')::date
GROUP BY day
ORDER BY day;

-- name: ListTopSearchesSinceAgg :many
-- Top terms by total count over the last @days complete days (current excluded).
SELECT query, SUM(search_count)::BIGINT AS search_count
FROM search_term_daily
WHERE day >= (NOW() AT TIME ZONE 'UTC')::date - @days::int
  AND day < (NOW() AT TIME ZONE 'UTC')::date
GROUP BY query
ORDER BY search_count DESC
LIMIT @lim::int;

-- name: DailySearchTermVolumeAgg :many
-- Per-day counts for the given terms over the last @days complete days.
SELECT query, to_char(day, 'YYYY-MM-DD') AS day, search_count
FROM search_term_daily
WHERE day >= (NOW() AT TIME ZONE 'UTC')::date - @days::int
  AND day < (NOW() AT TIME ZONE 'UTC')::date
  AND query = ANY(@terms::text[])
ORDER BY query, day;
