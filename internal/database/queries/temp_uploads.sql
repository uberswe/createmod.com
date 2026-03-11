-- name: CreateTempUpload :one
INSERT INTO temp_uploads (token, uploaded_by, filename, description, size, checksum, block_count, dim_x, dim_y, dim_z, mods, materials, minecraft_version, createmod_version, nbt_s3_key, image_s3_key, parsed_summary)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
RETURNING id, token, uploaded_by, filename, description, size, checksum, block_count, dim_x, dim_y, dim_z, mods, materials, minecraft_version, createmod_version, nbt_s3_key, image_s3_key, parsed_summary, created, updated;

-- name: GetTempUploadByToken :one
SELECT id, token, uploaded_by, filename, description, size, checksum, block_count, dim_x, dim_y, dim_z, mods, materials, minecraft_version, createmod_version, nbt_s3_key, image_s3_key, parsed_summary, created, updated
FROM temp_uploads
WHERE token = $1;

-- name: GetTempUploadByChecksum :one
SELECT id, token, uploaded_by, filename, description, size, checksum, block_count, dim_x, dim_y, dim_z, mods, materials, minecraft_version, createmod_version, nbt_s3_key, image_s3_key, parsed_summary, created, updated
FROM temp_uploads
WHERE checksum = $1
ORDER BY created DESC
LIMIT 1;

-- name: UpdateTempUpload :exec
UPDATE temp_uploads
SET filename = $2, description = $3, nbt_s3_key = $4, image_s3_key = $5, updated = NOW()
WHERE token = $1;

-- name: ClaimTempUpload :execrows
UPDATE temp_uploads
SET uploaded_by = $2, updated = NOW()
WHERE token = $1 AND uploaded_by = '';

-- name: DeleteTempUpload :exec
DELETE FROM temp_uploads WHERE token = $1;

-- name: DeleteExpiredTempUploads :execrows
DELETE FROM temp_uploads WHERE created < $1;

-- name: CreateTempUploadFile :one
INSERT INTO temp_upload_files (token, filename, description, size, checksum, block_count, dim_x, dim_y, dim_z, mods, materials, nbt_s3_key)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id, token, filename, description, size, checksum, block_count, dim_x, dim_y, dim_z, mods, materials, nbt_s3_key, created;

-- name: ListTempUploadFilesByToken :many
SELECT id, token, filename, description, size, checksum, block_count, dim_x, dim_y, dim_z, mods, materials, nbt_s3_key, created
FROM temp_upload_files
WHERE token = $1
ORDER BY created ASC;

-- name: GetTempUploadFileByID :one
SELECT id, token, filename, description, size, checksum, block_count, dim_x, dim_y, dim_z, mods, materials, nbt_s3_key, created
FROM temp_upload_files
WHERE id = $1;

-- name: DeleteTempUploadFile :exec
DELETE FROM temp_upload_files WHERE id = $1;

-- name: ListTempUploadsByUser :many
SELECT id, token, uploaded_by, filename, description, size, checksum,
       block_count, dim_x, dim_y, dim_z, mods, materials,
       minecraft_version, createmod_version, nbt_s3_key, image_s3_key,
       parsed_summary, created, updated
FROM temp_uploads
WHERE uploaded_by = $1
ORDER BY created DESC
LIMIT $2 OFFSET $3;

-- name: DeleteTempUploadFilesByToken :exec
DELETE FROM temp_upload_files WHERE token = $1;

-- name: ListExpiredUnclaimedTempUploads :many
SELECT id, token, nbt_s3_key, image_s3_key
FROM temp_uploads
WHERE uploaded_by = '' AND created < $1
ORDER BY created ASC
LIMIT $2;

-- name: DeleteExpiredUnclaimedTempUploads :execrows
DELETE FROM temp_uploads
WHERE uploaded_by = '' AND created < $1;
