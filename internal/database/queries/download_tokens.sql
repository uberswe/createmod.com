-- name: CreateDownloadToken :one
INSERT INTO download_tokens (token, name, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ConsumeDownloadToken :one
UPDATE download_tokens
SET used = true
WHERE token = $1 AND used = false AND expires_at > NOW()
RETURNING *;

-- name: CleanupExpiredDownloadTokens :exec
DELETE FROM download_tokens WHERE expires_at < NOW() - INTERVAL '1 hour';
