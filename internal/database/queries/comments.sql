-- name: GetCommentByID :one
SELECT * FROM comments WHERE id = $1;

-- name: ListCommentsBySchematic :many
SELECT c.*,
       u.username AS author_username,
       u.avatar AS author_avatar
FROM comments c
LEFT JOIN users u ON u.id = c.author_id
WHERE c.schematic_id = $1 AND c.approved = true
ORDER BY c.created ASC;

-- name: CountCommentsBySchematic :one
SELECT COUNT(*) FROM comments
WHERE schematic_id = $1 AND approved = true;

-- name: CreateComment :one
INSERT INTO comments (id, author_id, schematic_id, parent_id, content, published, approved, type)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ApproveComment :exec
UPDATE comments SET approved = true WHERE id = $1;

-- name: DeleteComment :exec
DELETE FROM comments WHERE id = $1;

-- name: DisapproveComment :exec
UPDATE comments SET approved = false WHERE id = $1;

-- name: CountUserComments :one
SELECT COUNT(*) FROM comments WHERE author_id = $1 AND approved = true;
