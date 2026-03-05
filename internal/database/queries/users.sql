-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 AND deleted IS NULL;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 AND deleted IS NULL;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE LOWER(username) = LOWER($1) AND deleted IS NULL;

-- name: CreateUser :one
INSERT INTO users (id, email, username, password_hash, old_password, avatar, verified)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET email = COALESCE(sqlc.narg('email'), email),
    username = COALESCE(sqlc.narg('username'), username),
    password_hash = COALESCE(sqlc.narg('password_hash'), password_hash),
    old_password = COALESCE(sqlc.narg('old_password'), old_password),
    avatar = COALESCE(sqlc.narg('avatar'), avatar),
    points = COALESCE(sqlc.narg('points'), points),
    verified = COALESCE(sqlc.narg('verified'), verified),
    is_admin = COALESCE(sqlc.narg('is_admin'), is_admin)
WHERE id = $1
RETURNING *;

-- name: SoftDeleteUser :exec
UPDATE users SET deleted = NOW() WHERE id = $1;

-- name: UpdateUserPoints :exec
UPDATE users SET points = $2 WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2, old_password = '' WHERE id = $1;

-- name: UpdateUserAvatar :exec
UPDATE users SET avatar = $2 WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users
WHERE deleted IS NULL
ORDER BY created DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users WHERE deleted IS NULL;

-- name: GetUserIsContributor :one
SELECT EXISTS(
    SELECT 1 FROM schematics
    WHERE author_id = $1 AND deleted IS NULL
    LIMIT 1
) AS is_contributor;

-- name: ListUsersForSitemap :many
SELECT id, username, updated FROM users
WHERE deleted IS NULL
ORDER BY updated DESC;
