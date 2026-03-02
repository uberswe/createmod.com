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
