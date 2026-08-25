-- name: CreateTempUploadImage :one
INSERT INTO temp_upload_images (token, filename, size, s3_key, sort_order, category)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, token, filename, size, s3_key, sort_order, category, moderation_status, created;

-- name: ListTempUploadImagesByToken :many
SELECT id, token, filename, size, s3_key, sort_order, category, moderation_status, created
FROM temp_upload_images
WHERE token = $1
ORDER BY sort_order ASC;

-- name: ListTempUploadImagesByTokenAndCategory :many
SELECT id, token, filename, size, s3_key, sort_order, category, moderation_status, created
FROM temp_upload_images
WHERE token = $1 AND category = $2
ORDER BY sort_order ASC;

-- name: UpdateTempUploadImageModerationStatus :exec
UPDATE temp_upload_images SET moderation_status = $2 WHERE id = $1;

-- name: GetTempUploadImageModerationStatus :one
-- Used by the file server to 404 temp images that aren't approved yet. (#1646)
SELECT moderation_status FROM temp_upload_images
WHERE token = $1 AND filename = $2
LIMIT 1;

-- name: DeleteTempUploadImage :exec
DELETE FROM temp_upload_images WHERE id = $1;

-- name: DeleteTempUploadImagesByToken :exec
DELETE FROM temp_upload_images WHERE token = $1;

-- name: CountTempUploadImagesByToken :one
SELECT COUNT(*)::int FROM temp_upload_images WHERE token = $1;
