-- name: GetSchematicViewCount :one
SELECT COALESCE(count, 0)::INTEGER AS view_count
FROM schematic_views
WHERE schematic_id = $1 AND period = 'total'
LIMIT 1;

-- name: UpsertSchematicView :exec
INSERT INTO schematic_views (id, schematic_id, period, type, count)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET count = schematic_views.count + 1;

-- name: GetSchematicDownloadCount :one
SELECT COUNT(*)::INTEGER AS download_count
FROM schematic_downloads
WHERE schematic_id = $1;

-- name: RecordSchematicDownload :exec
INSERT INTO schematic_downloads (id, schematic_id, user_id)
VALUES ($1, $2, $3);

-- name: GetSchematicRating :one
SELECT
    COALESCE(AVG(rating), 0)::REAL AS avg_rating,
    COUNT(*)::INTEGER AS rating_count
FROM schematic_ratings
WHERE schematic_id = $1;

-- name: UpsertSchematicRating :exec
INSERT INTO schematic_ratings (id, user_id, schematic_id, rating, rated_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (user_id, schematic_id) DO UPDATE SET
    rating = EXCLUDED.rating,
    rated_at = NOW();

-- name: UpsertSchematicViewCount :exec
INSERT INTO schematic_views (id, schematic_id, type, period, count)
VALUES ($1, $2, $3, $4, 1)
ON CONFLICT (schematic_id, type, period)
DO UPDATE SET count = schematic_views.count + 1, updated = NOW();

-- name: GetTotalViewCount :one
SELECT COALESCE(count, 0)::INTEGER AS total_count
FROM schematic_views
WHERE schematic_id = $1 AND type = '4' AND period = 'total'
LIMIT 1;

-- name: FetchRecentViewsBySchematic :many
SELECT schematic_id AS id, SUM(count)::REAL AS v
FROM schematic_views
WHERE type = '0' AND created > $1
GROUP BY schematic_id;

-- name: FetchTotalViewsBySchematic :many
SELECT schematic_id AS id, SUM(count)::REAL AS v
FROM schematic_views
WHERE type = '0'
GROUP BY schematic_id;

-- name: FetchRatingSumBySchematic :many
SELECT schematic_id AS id, SUM(rating)::REAL AS v
FROM schematic_ratings
GROUP BY schematic_id;

-- name: FetchRatingCountBySchematic :many
SELECT schematic_id AS id, COUNT(rating)::REAL AS v
FROM schematic_ratings
GROUP BY schematic_id;

-- name: FetchRecentDownloadsBySchematic :many
SELECT schematic_id AS id, COUNT(*)::REAL AS v
FROM schematic_downloads
WHERE created > $1
GROUP BY schematic_id;

-- name: FetchTotalDownloadsBySchematic :many
SELECT schematic_id AS id, COUNT(*)::REAL AS v
FROM schematic_downloads
GROUP BY schematic_id;
