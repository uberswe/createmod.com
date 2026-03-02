-- name: CreateSession :one
INSERT INTO sessions (id, user_id, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetSession :one
SELECT s.*, u.id AS "user.id", u.email AS "user.email", u.username AS "user.username",
       u.avatar AS "user.avatar", u.points AS "user.points", u.is_admin AS "user.is_admin",
       u.verified AS "user.verified"
FROM sessions s
JOIN users u ON u.id = s.user_id AND u.deleted IS NULL
WHERE s.id = $1 AND s.expires_at > NOW();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1;

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: CleanupExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < NOW();
