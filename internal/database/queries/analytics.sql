-- name: RecordSchematicEvent :exec
INSERT INTO schematic_events (schematic_id, event_type, event_value)
VALUES ($1, $2, $3);

-- name: HourlySchematicViewCounts :many
SELECT to_char(created, 'YYYY-MM-DD HH24') AS hour, SUM(count)::BIGINT AS total
FROM schematic_views
WHERE schematic_id = $1 AND type = '5' AND created > $2
GROUP BY hour
ORDER BY hour;

-- name: HourlySchematicDownloadCounts :many
SELECT to_char(created, 'YYYY-MM-DD HH24') AS hour, COUNT(*)::BIGINT AS total
FROM schematic_downloads
WHERE schematic_id = $1 AND created > $2
GROUP BY hour
ORDER BY hour;

-- name: HourlySchematicEventCounts :many
SELECT to_char(created, 'YYYY-MM-DD HH24') AS hour, SUM(event_value)::BIGINT AS total
FROM schematic_events
WHERE schematic_id = $1 AND event_type = $2 AND created > $3
GROUP BY hour
ORDER BY hour;

-- name: HourlySchematicEventAvg :many
SELECT to_char(created, 'YYYY-MM-DD HH24') AS hour, AVG(event_value)::BIGINT AS total
FROM schematic_events
WHERE schematic_id = $1 AND event_type = $2 AND created > $3
GROUP BY hour
ORDER BY hour;

-- name: HourlyUserAggregateViewCounts :many
SELECT to_char(sv.created, 'YYYY-MM-DD HH24') AS hour, SUM(sv.count)::BIGINT AS total
FROM schematic_views sv
JOIN schematics s ON s.id = sv.schematic_id
WHERE s.author_id = $1 AND sv.type = '5' AND sv.created > $2 AND s.deleted IS NULL
GROUP BY hour
ORDER BY hour;

-- name: HourlyUserAggregateDownloadCounts :many
SELECT to_char(sd.created, 'YYYY-MM-DD HH24') AS hour, COUNT(*)::BIGINT AS total
FROM schematic_downloads sd
JOIN schematics s ON s.id = sd.schematic_id
WHERE s.author_id = $1 AND sd.created > $2 AND s.deleted IS NULL
GROUP BY hour
ORDER BY hour;

-- name: HourlyUserAggregateEventCounts :many
SELECT to_char(se.created, 'YYYY-MM-DD HH24') AS hour, SUM(se.event_value)::BIGINT AS total
FROM schematic_events se
JOIN schematics s ON s.id = se.schematic_id
WHERE s.author_id = $1 AND se.event_type = $2 AND se.created > $3 AND s.deleted IS NULL
GROUP BY hour
ORDER BY hour;

-- name: HourlyUserAggregateEventAvg :many
SELECT to_char(se.created, 'YYYY-MM-DD HH24') AS hour, AVG(se.event_value)::BIGINT AS total
FROM schematic_events se
JOIN schematics s ON s.id = se.schematic_id
WHERE s.author_id = $1 AND se.event_type = $2 AND se.created > $3 AND s.deleted IS NULL
GROUP BY hour
ORDER BY hour;

-- name: ListSchematicStatsForUser :many
SELECT s.id, s.name, s.title, s.featured_image,
       COALESCE(sv.count, 0)::INTEGER AS views,
       COALESCE(dl.dl_count, 0)::INTEGER AS downloads,
       s.created
FROM schematics s
LEFT JOIN schematic_views sv ON sv.schematic_id = s.id AND sv.type = '4' AND sv.period = 'total'
LEFT JOIN (
    SELECT schematic_id, COUNT(*) AS dl_count FROM schematic_downloads GROUP BY schematic_id
) dl ON dl.schematic_id = s.id
WHERE s.author_id = $1 AND s.deleted IS NULL
ORDER BY s.created DESC
LIMIT $2 OFFSET $3;

-- name: CountUserSchematics :one
SELECT COUNT(*)::INTEGER AS total
FROM schematics
WHERE author_id = $1 AND deleted IS NULL;

-- name: GetSiteAvgVDRatio :one
SELECT CASE WHEN COALESCE(SUM(sv.count), 0) = 0 THEN 0
            ELSE COALESCE(SUM(dl.dl_count), 0)::REAL / SUM(sv.count)::REAL
       END AS ratio
FROM schematic_views sv
LEFT JOIN (
    SELECT schematic_id, COUNT(*) AS dl_count FROM schematic_downloads GROUP BY schematic_id
) dl ON dl.schematic_id = sv.schematic_id
WHERE sv.type = '4' AND sv.period = 'total';

-- name: DeleteOldSchematicEvents :execrows
DELETE FROM schematic_events WHERE created < $1;
