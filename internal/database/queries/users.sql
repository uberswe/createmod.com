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

-- name: ListAdminEmails :many
SELECT email FROM users WHERE is_admin = true AND deleted IS NULL;

-- name: GetUserByIDIncludingDeleted :one
SELECT * FROM users WHERE id = $1;

-- name: RestoreUser :exec
UPDATE users SET deleted = NULL WHERE id = $1;

-- name: ListUsersForAdmin :many
SELECT *
FROM users
WHERE
  (sqlc.arg('filter')::text = 'all'
     OR (sqlc.arg('filter')::text = 'active' AND deleted IS NULL)
     OR (sqlc.arg('filter')::text = 'deleted' AND deleted IS NOT NULL)
     OR (sqlc.arg('filter')::text = 'admin' AND is_admin = true AND deleted IS NULL))
  AND (sqlc.arg('search')::text = ''
     OR username ILIKE '%' || sqlc.arg('search')::text || '%'
     OR email ILIKE '%' || sqlc.arg('search')::text || '%')
ORDER BY created DESC
LIMIT $1 OFFSET $2;

-- name: CountUsersForAdmin :one
SELECT COUNT(*)
FROM users
WHERE
  (sqlc.arg('filter')::text = 'all'
     OR (sqlc.arg('filter')::text = 'active' AND deleted IS NULL)
     OR (sqlc.arg('filter')::text = 'deleted' AND deleted IS NOT NULL)
     OR (sqlc.arg('filter')::text = 'admin' AND is_admin = true AND deleted IS NULL))
  AND (sqlc.arg('search')::text = ''
     OR username ILIKE '%' || sqlc.arg('search')::text || '%'
     OR email ILIKE '%' || sqlc.arg('search')::text || '%');
