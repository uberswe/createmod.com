-- name: HourlySchematicStats :many
SELECT to_char(created, 'YYYY-MM-DD HH24') AS hour, COUNT(*) AS count
FROM schematics
WHERE created > $1 AND deleted IS NULL
GROUP BY hour
ORDER BY hour;

-- name: HourlyCommentStats :many
SELECT to_char(created, 'YYYY-MM-DD HH24') AS hour, COUNT(*) AS count
FROM comments
WHERE created > $1
GROUP BY hour
ORDER BY hour;

-- name: HourlyUserStats :many
SELECT to_char(created, 'YYYY-MM-DD HH24') AS hour, COUNT(*) AS count
FROM users
WHERE created > $1 AND deleted IS NULL
GROUP BY hour
ORDER BY hour;

-- name: MonthlyUserUploads :many
SELECT to_char(created, 'YYYY-MM') AS month, COUNT(*) AS count
FROM schematics
WHERE author_id = $1
  AND deleted IS NULL
  AND created > NOW() - make_interval(months => $2::int)
GROUP BY month
ORDER BY month;

-- name: MonthlyUserDownloads :many
SELECT to_char(sd.created, 'YYYY-MM') AS month, COUNT(*) AS count
FROM schematic_downloads sd
JOIN schematics s ON s.id = sd.schematic_id
WHERE s.author_id = $1
  AND s.deleted IS NULL
  AND sd.created > NOW() - make_interval(months => $2::int)
GROUP BY month
ORDER BY month;

-- name: MonthlyUserViews :many
SELECT to_char(sv.created, 'YYYY-MM') AS month, SUM(sv.count)::BIGINT AS count
FROM schematic_views sv
JOIN schematics s ON s.id = sv.schematic_id
WHERE s.author_id = $1
  AND s.deleted IS NULL
  AND sv.type = '0'
  AND sv.created > NOW() - make_interval(months => $2::int)
GROUP BY month
ORDER BY month;
