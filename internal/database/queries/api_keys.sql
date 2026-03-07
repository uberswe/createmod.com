-- name: GetAPIKeyByLast8 :one
SELECT * FROM api_keys WHERE last8 = $1;

-- name: ListAPIKeysByUser :many
SELECT * FROM api_keys WHERE user_id = $1 ORDER BY created DESC;

-- name: CreateAPIKey :one
INSERT INTO api_keys (id, user_id, key_hash, label, last8)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: DeleteAPIKey :exec
DELETE FROM api_keys WHERE id = $1 AND user_id = $2;

-- name: LogAPIKeyUsage :exec
INSERT INTO api_key_usage (id, api_key_id, endpoint)
VALUES ($1, $2, $3);
