-- name: CreateTempUploadImage :one
INSERT INTO temp_upload_images (token, filename, size, s3_key, sort_order)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, token, filename, size, s3_key, sort_order, created;

-- name: ListTempUploadImagesByToken :many
SELECT id, token, filename, size, s3_key, sort_order, created
FROM temp_upload_images
WHERE token = $1
ORDER BY sort_order ASC;

-- name: DeleteTempUploadImage :exec
DELETE FROM temp_upload_images WHERE id = $1;

-- name: DeleteTempUploadImagesByToken :exec
DELETE FROM temp_upload_images WHERE token = $1;

-- name: CountTempUploadImagesByToken :one
SELECT COUNT(*)::int FROM temp_upload_images WHERE token = $1;
