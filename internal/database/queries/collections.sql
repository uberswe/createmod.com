-- name: GetCollectionByID :one
SELECT * FROM collections WHERE id = $1 AND deleted = '';

-- name: GetCollectionBySlug :one
SELECT * FROM collections WHERE slug = $1 AND deleted = '';

-- name: ListCollections :many
SELECT * FROM collections
WHERE deleted = '' AND published = true
ORDER BY updated DESC
LIMIT $1 OFFSET $2;

-- name: ListCollectionsByAuthor :many
SELECT * FROM collections
WHERE author_id = $1 AND deleted = ''
ORDER BY updated DESC;

-- name: ListFeaturedCollections :many
SELECT * FROM collections
WHERE deleted = '' AND published = true AND featured = true
ORDER BY updated DESC
LIMIT $1;

-- name: CreateCollection :one
INSERT INTO collections (id, author_id, title, name, slug, description, banner_url, published, video)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateCollection :one
UPDATE collections SET
    title = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    banner_url = COALESCE(sqlc.narg('banner_url'), banner_url),
    collage_url = COALESCE(sqlc.narg('collage_url'), collage_url),
    featured = COALESCE(sqlc.narg('featured'), featured),
    published_at = CASE
        WHEN COALESCE(sqlc.narg('published'), published) = true AND published = false THEN NOW()
        ELSE published_at
    END,
    published = COALESCE(sqlc.narg('published'), published),
    video = COALESCE(sqlc.narg('video'), video),
    updated = NOW()
WHERE id = $1
RETURNING *;

-- name: SoftDeleteCollection :exec
UPDATE collections SET deleted = 'deleted' WHERE id = $1;

-- name: SoftDeleteCollectionsByAuthor :exec
UPDATE collections SET deleted = 'deleted' WHERE author_id = $1 AND deleted = '';

-- name: RestoreCollectionsByAuthor :exec
UPDATE collections SET deleted = '' WHERE author_id = $1 AND deleted = 'deleted';

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

-- name: ListPublishedCollections :many
SELECT * FROM collections
WHERE deleted = '' AND published = true
ORDER BY updated DESC
LIMIT $1 OFFSET $2;

-- name: ListCollectionsPublishedSince :many
SELECT * FROM collections
WHERE deleted = '' AND published = true
  AND published_at >= @since AND published_at < @until
ORDER BY published_at DESC;

-- name: UpdateCollectionCollageURL :exec
UPDATE collections SET collage_url = $2 WHERE id = $1;

-- name: ListCollectionsForSitemap :many
SELECT id, slug, updated FROM collections
WHERE deleted = '' AND published = true
ORDER BY updated DESC;

-- name: ListCollectionsForAdmin :many
SELECT * FROM collections
WHERE
  CASE
    WHEN @filter::text = 'published' THEN deleted = '' AND published = true
    WHEN @filter::text = 'unpublished' THEN deleted = '' AND published = false
    WHEN @filter::text = 'deleted' THEN deleted != ''
    ELSE deleted = ''
  END
ORDER BY updated DESC
LIMIT $1 OFFSET $2;

-- name: CountCollectionsForAdmin :one
SELECT COUNT(*) FROM collections
WHERE
  CASE
    WHEN @filter::text = 'published' THEN deleted = '' AND published = true
    WHEN @filter::text = 'unpublished' THEN deleted = '' AND published = false
    WHEN @filter::text = 'deleted' THEN deleted != ''
    ELSE deleted = ''
  END;
