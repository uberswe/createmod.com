-- name: GetSchematicByID :one
SELECT * FROM schematics WHERE id = $1 AND deleted IS NULL;

-- name: GetSchematicByName :one
SELECT * FROM schematics
WHERE name = $1
  AND deleted IS NULL
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
LIMIT 1;

-- name: ListApprovedSchematics :many
SELECT * FROM schematics
WHERE deleted IS NULL
  AND moderated = true
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
ORDER BY created DESC
LIMIT $1 OFFSET $2;

-- name: CountApprovedSchematics :one
SELECT COUNT(*) FROM schematics
WHERE deleted IS NULL
  AND moderated = true
  AND (scheduled_at IS NULL OR scheduled_at <= NOW());

-- name: ListSchematicsByAuthor :many
SELECT * FROM schematics
WHERE author_id = $1
  AND deleted IS NULL
  AND moderated = true
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
ORDER BY created DESC
LIMIT $2 OFFSET $3;

-- name: ListSchematicsByAuthorExcluding :many
SELECT * FROM schematics
WHERE author_id = $1
  AND id != $2
  AND deleted IS NULL
  AND moderated = true
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
ORDER BY created DESC
LIMIT $3;

-- name: ListSchematicsByIDs :many
SELECT * FROM schematics
WHERE id = ANY($1::text[])
  AND deleted IS NULL;

-- name: ListFeaturedSchematics :many
SELECT * FROM schematics
WHERE deleted IS NULL
  AND moderated = true
  AND featured = true
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
ORDER BY created DESC
LIMIT $1;

-- name: ListAllApprovedSchematicsForIndex :many
SELECT * FROM schematics
WHERE deleted IS NULL
  AND moderated = true
ORDER BY created DESC;

-- name: CreateSchematic :one
INSERT INTO schematics (
    id, author_id, name, title, description, excerpt, content,
    postdate, detected_language, featured_image, gallery, schematic_file,
    video, has_dependencies, dependencies, createmod_version_id,
    minecraft_version_id, block_count, dim_x, dim_y, dim_z,
    materials, mods, paid, moderated, type, status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11, $12,
    $13, $14, $15, $16,
    $17, $18, $19, $20, $21,
    $22, $23, $24, $25, $26, $27
)
RETURNING *;

-- name: UpdateSchematic :one
UPDATE schematics SET
    title = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    excerpt = COALESCE(sqlc.narg('excerpt'), excerpt),
    content = COALESCE(sqlc.narg('content'), content),
    featured_image = COALESCE(sqlc.narg('featured_image'), featured_image),
    gallery = COALESCE(sqlc.narg('gallery'), gallery),
    video = COALESCE(sqlc.narg('video'), video),
    has_dependencies = COALESCE(sqlc.narg('has_dependencies'), has_dependencies),
    dependencies = COALESCE(sqlc.narg('dependencies'), dependencies),
    createmod_version_id = COALESCE(sqlc.narg('createmod_version_id'), createmod_version_id),
    minecraft_version_id = COALESCE(sqlc.narg('minecraft_version_id'), minecraft_version_id),
    ai_description = COALESCE(sqlc.narg('ai_description'), ai_description),
    moderated = COALESCE(sqlc.narg('moderated'), moderated),
    moderation_reason = COALESCE(sqlc.narg('moderation_reason'), moderation_reason),
    blacklisted = COALESCE(sqlc.narg('blacklisted'), blacklisted),
    featured = COALESCE(sqlc.narg('featured'), featured),
    scheduled_at = COALESCE(sqlc.narg('scheduled_at'), scheduled_at),
    block_count = COALESCE(sqlc.narg('block_count'), block_count),
    dim_x = COALESCE(sqlc.narg('dim_x'), dim_x),
    dim_y = COALESCE(sqlc.narg('dim_y'), dim_y),
    dim_z = COALESCE(sqlc.narg('dim_z'), dim_z),
    materials = COALESCE(sqlc.narg('materials'), materials),
    mods = COALESCE(sqlc.narg('mods'), mods),
    paid = COALESCE(sqlc.narg('paid'), paid),
    external_url = COALESCE(sqlc.narg('external_url'), external_url)
WHERE id = $1
RETURNING *;

-- name: SoftDeleteSchematic :exec
UPDATE schematics SET deleted = NOW(), deleted_at = NOW() WHERE id = $1;

-- name: UpdateSchematicViews :exec
UPDATE schematics SET views = $2 WHERE id = $1;

-- name: UpdateSchematicDownloads :exec
UPDATE schematics SET downloads = $2 WHERE id = $1;

-- name: IncrementSchematicDownloads :exec
UPDATE schematics SET downloads = downloads + 1 WHERE id = $1;

-- name: GetSchematicCategoryIDs :many
SELECT category_id FROM schematics_categories WHERE schematic_id = $1;

-- name: GetSchematicTagIDs :many
SELECT tag_id FROM schematics_tags WHERE schematic_id = $1;

-- name: SetSchematicCategories :exec
DELETE FROM schematics_categories WHERE schematic_id = $1;

-- name: AddSchematicCategory :exec
INSERT INTO schematics_categories (schematic_id, category_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: SetSchematicTags :exec
DELETE FROM schematics_tags WHERE schematic_id = $1;

-- name: AddSchematicTag :exec
INSERT INTO schematics_tags (schematic_id, tag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: ListApprovedSchematicsWithVideo :many
SELECT * FROM schematics
WHERE deleted IS NULL
  AND moderated = true
  AND video != ''
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
ORDER BY created DESC
LIMIT $1 OFFSET $2;

-- name: ListRandomApprovedSchematics :many
SELECT * FROM schematics
WHERE deleted IS NULL
  AND moderated = true
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
ORDER BY RANDOM()
LIMIT $1;

-- name: ListSchematicsByCategoryIDs :many
SELECT DISTINCT s.* FROM schematics s
JOIN schematics_categories sc ON sc.schematic_id = s.id
WHERE sc.category_id = ANY($1::text[])
  AND s.id != ALL($2::text[])
  AND s.deleted IS NULL
  AND s.moderated = true
  AND (s.scheduled_at IS NULL OR s.scheduled_at <= NOW())
ORDER BY s.views DESC
LIMIT $3;

-- name: ListHighestRatedSchematics :many
SELECT s.* FROM schematics s
JOIN schematic_ratings sr ON sr.schematic_id = s.id
WHERE s.deleted IS NULL
  AND s.moderated = true
  AND (s.scheduled_at IS NULL OR s.scheduled_at <= NOW())
GROUP BY s.id
HAVING COUNT(sr.rating) > 0
ORDER BY AVG(sr.rating) DESC, COUNT(sr.rating) DESC
LIMIT $1 OFFSET $2;

-- name: ListSchematicsForSitemap :many
SELECT id, name, updated FROM schematics
WHERE deleted IS NULL
  AND moderated = true
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
ORDER BY updated DESC;

-- name: CountSchematicsByAuthor :one
SELECT COUNT(*) FROM schematics
WHERE author_id = $1
  AND deleted IS NULL
  AND moderated = true;

-- name: GetSchematicByChecksum :one
SELECT nh.schematic_id FROM nbt_hashes nh
JOIN schematics s ON s.id = nh.schematic_id
WHERE nh.hash = $1
  AND s.moderated = true
  AND s.deleted IS NULL
LIMIT 1;

-- name: UpdateSchematicName :exec
UPDATE schematics SET name = $2 WHERE id = $1;

-- name: ListSchematicsByNamePattern :many
SELECT * FROM schematics
WHERE name LIKE $1
  AND deleted IS NULL
LIMIT $2;

-- name: ListSchematicsForAdmin :many
SELECT * FROM schematics
WHERE
  CASE
    WHEN @filter::text = 'pending' THEN moderated = false AND deleted IS NULL
    WHEN @filter::text = 'moderated' THEN moderated = true AND deleted IS NULL
    WHEN @filter::text = 'deleted' THEN deleted IS NOT NULL
    ELSE true
  END
ORDER BY created DESC
LIMIT $1 OFFSET $2;

-- name: CountSchematicsForAdmin :one
SELECT COUNT(*) FROM schematics
WHERE
  CASE
    WHEN @filter::text = 'pending' THEN moderated = false AND deleted IS NULL
    WHEN @filter::text = 'moderated' THEN moderated = true AND deleted IS NULL
    WHEN @filter::text = 'deleted' THEN deleted IS NOT NULL
    ELSE true
  END;

-- name: GetSchematicByIDAdmin :one
SELECT * FROM schematics WHERE id = $1;
