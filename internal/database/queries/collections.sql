-- name: GetCollectionByID :one
SELECT * FROM collections WHERE id = $1 AND deleted = '';

-- name: GetCollectionBySlug :one
SELECT * FROM collections WHERE slug = $1 AND deleted = '';

-- name: ListCollections :many
SELECT * FROM collections
WHERE deleted = '' AND published = true
ORDER BY created DESC
LIMIT $1 OFFSET $2;

-- name: ListCollectionsByAuthor :many
SELECT * FROM collections
WHERE author_id = $1 AND deleted = ''
ORDER BY created DESC;

-- name: ListFeaturedCollections :many
SELECT * FROM collections
WHERE deleted = '' AND published = true AND featured = true
ORDER BY created DESC
LIMIT $1;

-- name: CreateCollection :one
INSERT INTO collections (id, author_id, title, name, slug, description, banner_url, published)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateCollection :one
UPDATE collections SET
    title = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    banner_url = COALESCE(sqlc.narg('banner_url'), banner_url),
    featured = COALESCE(sqlc.narg('featured'), featured),
    published = COALESCE(sqlc.narg('published'), published)
WHERE id = $1
RETURNING *;

-- name: SoftDeleteCollection :exec
UPDATE collections SET deleted = 'deleted' WHERE id = $1;

-- name: GetCollectionSchematicIDs :many
SELECT schematic_id FROM collections_schematics
WHERE collection_id = $1
ORDER BY position;

-- name: AddSchematicToCollection :exec
INSERT INTO collections_schematics (collection_id, schematic_id, position)
VALUES ($1, $2, $3)
ON CONFLICT (collection_id, schematic_id) DO UPDATE SET position = $3;

-- name: RemoveSchematicFromCollection :exec
DELETE FROM collections_schematics
WHERE collection_id = $1 AND schematic_id = $2;

-- name: ClearCollectionSchematics :exec
DELETE FROM collections_schematics WHERE collection_id = $1;

-- name: IncrementCollectionViews :exec
UPDATE collections SET views = views + 1 WHERE id = $1;

-- name: CountUserCollections :one
SELECT COUNT(*) FROM collections WHERE author_id = $1 AND deleted = '';
