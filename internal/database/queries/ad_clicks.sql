-- name: UpsertAdClick :exec
INSERT INTO ad_clicks (ad_unit, dest, period, count, created, updated)
VALUES ($1, $2, $3, 1, now(), now())
ON CONFLICT (ad_unit, dest, period)
DO UPDATE SET count = ad_clicks.count + 1, updated = now();

-- name: ListDailyAdClicks :many
SELECT ad_unit, dest, period, count
FROM ad_clicks
WHERE period LIKE '________'
ORDER BY period DESC, ad_unit;

-- name: ListMonthlyAdClicks :many
SELECT ad_unit, dest, period, count
FROM ad_clicks
WHERE period LIKE '______'
ORDER BY period DESC, ad_unit;

-- name: RollupDailyToMonthly :exec
INSERT INTO ad_clicks (ad_unit, dest, period, count, created, updated)
SELECT d.ad_unit, d.dest, LEFT(d.period, 6), SUM(d.count), now(), now()
FROM ad_clicks d
WHERE d.period LIKE '________' AND d.period < $1
GROUP BY d.ad_unit, d.dest, LEFT(d.period, 6)
ON CONFLICT (ad_unit, dest, period)
DO UPDATE SET count = ad_clicks.count + EXCLUDED.count, updated = now();

-- name: DeleteOldDailyAdClicks :exec
DELETE FROM ad_clicks
WHERE period LIKE '________' AND period < $1;
