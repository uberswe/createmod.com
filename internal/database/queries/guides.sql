-- name: GetGuideByID :one
SELECT * FROM guides WHERE id = $1;

-- name: GetGuideBySlug :one
SELECT * FROM guides WHERE slug = $1;

-- name: ListGuides :many
SELECT * FROM guides ORDER BY created DESC LIMIT $1 OFFSET $2;

-- name: CreateGuide :one
INSERT INTO guides (id, author_id, title, description, content, slug, upload_link, banner_url)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateGuide :one
UPDATE guides SET
    title = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    content = COALESCE(sqlc.narg('content'), content),
    upload_link = COALESCE(sqlc.narg('upload_link'), upload_link),
    banner_url = COALESCE(sqlc.narg('banner_url'), banner_url)
WHERE id = $1
RETURNING *;

-- name: DeleteGuide :exec
DELETE FROM guides WHERE id = $1;

-- name: CountUserGuides :one
SELECT COUNT(*) FROM guides WHERE author_id = $1;

-- name: IncrementGuideViews :exec
UPDATE guides SET views = views + 1 WHERE id = $1;

-- name: ListGuidesForSitemap :many
SELECT id, slug, updated FROM guides ORDER BY updated DESC;
