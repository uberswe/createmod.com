-- name: GetCommentByID :one
SELECT * FROM comments WHERE id = $1;

-- name: ListCommentsBySchematic :many
SELECT c.*,
       u.username AS author_username,
       u.avatar AS author_avatar
FROM comments c
LEFT JOIN users u ON u.id = c.author_id
WHERE c.schematic_id = $1 AND c.approved = true AND c.deleted IS NULL
ORDER BY c.created ASC;

-- name: CountCommentsBySchematic :one
SELECT COUNT(*) FROM comments
WHERE schematic_id = $1 AND approved = true AND deleted IS NULL;

-- name: CreateComment :one
INSERT INTO comments (id, author_id, schematic_id, parent_id, content, published, approved, type)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ApproveComment :exec
UPDATE comments SET approved = true WHERE id = $1;

-- name: DeleteComment :exec
UPDATE comments SET deleted = NOW() WHERE id = $1;

-- name: SoftDeleteCommentsByAuthor :exec
UPDATE comments SET deleted = NOW() WHERE author_id = $1 AND deleted IS NULL;

-- name: RestoreCommentsByAuthor :exec
UPDATE comments SET deleted = NULL WHERE author_id = $1 AND deleted IS NOT NULL;

-- name: RestoreComment :exec
UPDATE comments SET deleted = NULL WHERE id = $1;

-- name: HardDeleteComment :exec
DELETE FROM comments WHERE id = $1;

-- name: DisapproveComment :exec
UPDATE comments SET approved = false WHERE id = $1;

-- name: CountUserComments :one
SELECT COUNT(*) FROM comments WHERE author_id = $1 AND approved = true AND deleted IS NULL;

-- name: ListCommentsForAdmin :many
SELECT c.*,
       u.username AS author_username,
       u.avatar AS author_avatar,
       s.name AS schematic_name,
       s.title AS schematic_title
FROM comments c
LEFT JOIN users u ON u.id = c.author_id
LEFT JOIN schematics s ON s.id = c.schematic_id
WHERE
  (sqlc.arg('filter')::text = 'all'
     OR (sqlc.arg('filter')::text = 'active' AND c.deleted IS NULL)
     OR (sqlc.arg('filter')::text = 'deleted' AND c.deleted IS NOT NULL)
     OR (sqlc.arg('filter')::text = 'unapproved' AND c.approved = false AND c.deleted IS NULL))
  AND (sqlc.arg('search')::text = ''
     OR c.content ILIKE '%' || sqlc.arg('search')::text || '%'
     OR u.username ILIKE '%' || sqlc.arg('search')::text || '%')
ORDER BY c.created DESC
LIMIT $1 OFFSET $2;

-- name: CountCommentsForAdmin :one
SELECT COUNT(*)
FROM comments c
LEFT JOIN users u ON u.id = c.author_id
WHERE
  (sqlc.arg('filter')::text = 'all'
     OR (sqlc.arg('filter')::text = 'active' AND c.deleted IS NULL)
     OR (sqlc.arg('filter')::text = 'deleted' AND c.deleted IS NOT NULL)
     OR (sqlc.arg('filter')::text = 'unapproved' AND c.approved = false AND c.deleted IS NULL))
  AND (sqlc.arg('search')::text = ''
     OR c.content ILIKE '%' || sqlc.arg('search')::text || '%'
     OR u.username ILIKE '%' || sqlc.arg('search')::text || '%');
