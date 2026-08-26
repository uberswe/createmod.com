-- name: GetSchematicByID :one
SELECT * FROM schematics WHERE id = $1 AND moderation_state != 'deleted';

-- name: GetSchematicByName :one
SELECT * FROM schematics
WHERE name = $1
  AND moderation_state != 'deleted'
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
LIMIT 1;

-- name: GetSchematicByShortCode :one
SELECT * FROM schematics
WHERE short_code = $1
  AND moderation_state != 'deleted'
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
LIMIT 1;

-- name: SetSchematicShortCode :exec
UPDATE schematics SET short_code = $2 WHERE id = $1;

-- name: ShortCodeExists :one
SELECT EXISTS(SELECT 1 FROM schematics WHERE short_code = $1) AS exists;

-- name: ListApprovedSchematics :many
SELECT * FROM schematics
WHERE moderation_state IN ('published', 'approved')
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
ORDER BY created DESC
LIMIT $1 OFFSET $2;

-- name: CountApprovedSchematics :one
SELECT COUNT(*) FROM schematics
WHERE moderation_state IN ('published', 'approved')
  AND (scheduled_at IS NULL OR scheduled_at <= NOW());

-- name: ListSchematicsByAuthor :many
SELECT * FROM schematics
WHERE author_id = $1
  AND moderation_state IN ('published', 'approved')
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
ORDER BY created DESC
LIMIT $2 OFFSET $3;

-- name: ListSchematicsByAuthorExcluding :many
SELECT * FROM schematics
WHERE author_id = $1
  AND id != $2
  AND moderation_state IN ('published', 'approved')
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
ORDER BY created DESC
LIMIT $3;

-- name: ListSchematicsByIDs :many
SELECT * FROM schematics
WHERE id = ANY($1::text[])
  AND moderation_state != 'deleted';

-- name: ListFeaturedSchematics :many
SELECT * FROM schematics
WHERE moderation_state IN ('published', 'approved')
  AND featured = true
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
ORDER BY created DESC
LIMIT $1;

-- name: ListAllApprovedSchematicsForIndex :many
SELECT * FROM schematics
WHERE moderation_state IN ('published', 'approved')
ORDER BY created DESC;

-- name: CreateSchematic :one
INSERT INTO schematics (
    id, author_id, name, title, description, excerpt, content,
    postdate, detected_language, featured_image, gallery, schematic_file,
    video, has_dependencies, dependencies, createmod_version_id,
    minecraft_version_id, block_count, dim_x, dim_y, dim_z,
    materials, mods, paid, moderation_state, type, status, rotation_images, short_code, rotation_disabled,
    source_format, original_file
) VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11, $12,
    $13, $14, $15, $16,
    $17, $18, $19, $20, $21,
    $22, $23, $24, $25, $26, $27, $28, $29, $30,
    $31, $32
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
    rotation_images = COALESCE(sqlc.narg('rotation_images'), rotation_images),
    video = COALESCE(sqlc.narg('video'), video),
    has_dependencies = COALESCE(sqlc.narg('has_dependencies'), has_dependencies),
    dependencies = COALESCE(sqlc.narg('dependencies'), dependencies),
    createmod_version_id = COALESCE(sqlc.narg('createmod_version_id'), createmod_version_id),
    minecraft_version_id = COALESCE(sqlc.narg('minecraft_version_id'), minecraft_version_id),
    ai_description = COALESCE(sqlc.narg('ai_description'), ai_description),
    moderation_state = COALESCE(sqlc.narg('moderation_state'), moderation_state),
    moderation_reason = COALESCE(sqlc.narg('moderation_reason'), moderation_reason),
    featured = COALESCE(sqlc.narg('featured'), featured),
    scheduled_at = COALESCE(sqlc.narg('scheduled_at'), scheduled_at),
    block_count = COALESCE(sqlc.narg('block_count'), block_count),
    dim_x = COALESCE(sqlc.narg('dim_x'), dim_x),
    dim_y = COALESCE(sqlc.narg('dim_y'), dim_y),
    dim_z = COALESCE(sqlc.narg('dim_z'), dim_z),
    materials = COALESCE(sqlc.narg('materials'), materials),
    mods = COALESCE(sqlc.narg('mods'), mods),
    paid = COALESCE(sqlc.narg('paid'), paid),
    external_url = COALESCE(sqlc.narg('external_url'), external_url),
    schematic_file = COALESCE(sqlc.narg('schematic_file'), schematic_file),
    short_code = COALESCE(sqlc.narg('short_code'), short_code),
    created = COALESCE(sqlc.narg('created'), created),
    rotation_disabled = COALESCE(sqlc.narg('rotation_disabled'), rotation_disabled),
    held_images = COALESCE(sqlc.narg('held_images'), held_images),
    removed_images = COALESCE(sqlc.narg('removed_images'), removed_images),
    moderation_resubmit_count = COALESCE(sqlc.narg('moderation_resubmit_count'), moderation_resubmit_count),
    modified = NOW()
WHERE id = $1
RETURNING *;

-- name: SetModerationState :exec
UPDATE schematics SET moderation_state = $2, moderation_reason = $3, updated = NOW() WHERE id = $1;

-- name: SoftDeleteSchematic :exec
UPDATE schematics SET deleted = NOW(), deleted_at = NOW(), moderation_state = 'deleted' WHERE id = $1;

-- name: SoftDeleteSchematicsByAuthor :exec
UPDATE schematics SET deleted = NOW(), deleted_at = NOW(), moderation_state = 'deleted'
WHERE author_id = $1 AND deleted IS NULL;

-- name: RestoreSchematicsByAuthor :exec
UPDATE schematics SET deleted = NULL, deleted_at = NULL, moderation_state = 'approved'
WHERE author_id = $1 AND deleted IS NOT NULL;

-- name: AddHeldImages :exec
-- Atomically merge filenames into held_images (deduped), so concurrent image
-- moderation goroutines (gallery vs featured) never clobber each other. (#1646)
UPDATE schematics
SET held_images = ARRAY(SELECT DISTINCT unnest(held_images || @filenames::text[])),
    modified = NOW()
WHERE id = @id;

-- name: ReassignFeaturedIfHeld :exec
-- If the featured image is now held or removed, fall back to the first
-- still-visible gallery image (or clear it). Atomic and idempotent. (#1646)
UPDATE schematics
SET featured_image = COALESCE(
      (SELECT g FROM unnest(gallery) AS g
       WHERE g <> ALL(held_images) AND g <> ALL(removed_images)
       LIMIT 1), ''),
    modified = NOW()
WHERE id = @id
  AND featured_image <> ''
  AND (featured_image = ANY(held_images) OR featured_image = ANY(removed_images));

-- name: ListModerationQueue :many
-- Schematics needing a moderator's attention: policy-flagged, with one or more
-- held images awaiting an approve/remove decision, or where the author asked a
-- human to re-check an automated outcome. (#1646)
SELECT * FROM schematics
WHERE deleted IS NULL
  AND (moderation_state = 'flagged'
       OR human_review_requested = true
       OR COALESCE(array_length(held_images, 1), 0) > 0)
ORDER BY created ASC
LIMIT $1 OFFSET $2;

-- name: CountModerationQueue :one
SELECT COUNT(*) FROM schematics
WHERE deleted IS NULL
  AND (moderation_state = 'flagged'
       OR human_review_requested = true
       OR COALESCE(array_length(held_images, 1), 0) > 0);

-- name: SetModerationReviewedBy :exec
UPDATE schematics SET moderation_reviewed_by = $2, updated = NOW() WHERE id = $1;

-- name: SetHumanReviewRequested :exec
UPDATE schematics SET human_review_requested = $2, updated = NOW() WHERE id = $1;

-- name: ListStuckAutoReview :many
-- Schematics stranded in auto_review past the cutoff: their async moderation
-- job never ran (legacy rows from the 026 migration) or was discarded. The
-- auto_review backfill re-enqueues the moderation check for each. (#1646)
SELECT * FROM schematics
WHERE deleted IS NULL
  AND moderation_state = 'auto_review'
  AND created < @older_than
ORDER BY created ASC
LIMIT @lim;

-- name: ApproveHeldImage :exec
-- Un-hold an image: it becomes visible to everyone again. (#1646)
UPDATE schematics
SET held_images = array_remove(held_images, @filename::text), modified = NOW()
WHERE id = @id;

-- name: RemoveHeldImage :exec
-- A moderator removed a held image after review: drop it from held and record it
-- in removed_images (never rendered again). (#1646)
UPDATE schematics
SET held_images = array_remove(held_images, @filename::text),
    removed_images = ARRAY(SELECT DISTINCT unnest(removed_images || ARRAY[@filename::text])),
    modified = NOW()
WHERE id = @id;

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
WHERE moderation_state IN ('published', 'approved')
  AND video != ''
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
ORDER BY created DESC
LIMIT $1 OFFSET $2;

-- name: ListRandomApprovedSchematics :many
SELECT * FROM schematics
WHERE moderation_state IN ('published', 'approved')
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
ORDER BY RANDOM()
LIMIT $1;

-- name: ListSchematicsByCategoryIDs :many
SELECT DISTINCT s.* FROM schematics s
JOIN schematics_categories sc ON sc.schematic_id = s.id
WHERE sc.category_id = ANY($1::text[])
  AND s.id != ALL($2::text[])
  AND s.moderation_state IN ('published', 'approved')
  AND (s.scheduled_at IS NULL OR s.scheduled_at <= NOW())
ORDER BY s.views DESC
LIMIT $3;

-- name: ListHighestRatedSchematics :many
SELECT * FROM schematics
WHERE moderation_state IN ('published', 'approved')
  AND rating_count > 0
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
ORDER BY avg_rating DESC, rating_count DESC
LIMIT $1 OFFSET $2;

-- name: UpdateSchematicTrendingScore :exec
UPDATE schematics SET trending_score = $2 WHERE id = $1;

-- name: UpdateSchematicRatingAggregates :exec
UPDATE schematics SET avg_rating = $2, rating_count = $3 WHERE id = $1;

-- name: RefreshSchematicRatingAggregates :exec
UPDATE schematics SET
  avg_rating = COALESCE(sub.avg_r, 0),
  rating_count = COALESCE(sub.cnt, 0)
FROM (SELECT AVG(rating)::REAL AS avg_r, COUNT(*)::INTEGER AS cnt FROM schematic_ratings WHERE schematic_id = $1 AND deleted IS NULL AND rating BETWEEN 1 AND 5) sub
WHERE schematics.id = $1;

-- name: ListSchematicsForSitemap :many
SELECT id, name, updated FROM schematics
WHERE moderation_state IN ('published', 'approved')
  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
ORDER BY updated DESC;

-- name: CountSchematicsByAuthor :one
SELECT COUNT(*) FROM schematics
WHERE author_id = $1
  AND moderation_state IN ('published', 'approved');

-- name: CountSoftDeletedByAuthor :one
SELECT COUNT(*) FROM schematics
WHERE author_id = $1
  AND deleted IS NOT NULL;

-- name: GetSchematicByChecksum :one
SELECT nh.schematic_id FROM nbt_hashes nh
JOIN schematics s ON s.id = nh.schematic_id
WHERE nh.hash = $1
  AND s.moderation_state IN ('published', 'approved')
  AND s.deleted IS NULL
LIMIT 1;

-- name: SchematicNameExists :one
SELECT EXISTS(
    SELECT 1 FROM schematics WHERE name = $1 AND moderation_state != 'deleted'
) AS exists;

-- name: UpdateSchematicName :exec
UPDATE schematics SET name = $2 WHERE id = $1;

-- name: ListSchematicsByNamePattern :many
SELECT * FROM schematics
WHERE name LIKE $1
  AND moderation_state != 'deleted'
LIMIT $2;

-- name: ListSchematicsForAdmin :many
SELECT * FROM schematics
WHERE
  CASE
    WHEN @filter::text = 'pending' THEN moderation_state IN ('auto_review', 'flagged')
    WHEN @filter::text = 'published' THEN moderation_state IN ('published', 'approved')
    WHEN @filter::text = 'flagged' THEN moderation_state = 'flagged'
    WHEN @filter::text = 'rejected' THEN moderation_state = 'rejected'
    WHEN @filter::text = 'deleted' THEN moderation_state = 'deleted'
    WHEN @filter::text = 'private' THEN moderation_state NOT IN ('published', 'approved')
    ELSE true
  END
  AND (@search::text = ''
     OR title ILIKE '%' || @search::text || '%'
     OR name ILIKE '%' || @search::text || '%'
     OR id = @search::text)
ORDER BY created DESC
LIMIT $1 OFFSET $2;

-- name: CountSchematicsForAdmin :one
SELECT COUNT(*) FROM schematics
WHERE
  CASE
    WHEN @filter::text = 'pending' THEN moderation_state IN ('auto_review', 'flagged')
    WHEN @filter::text = 'published' THEN moderation_state IN ('published', 'approved')
    WHEN @filter::text = 'flagged' THEN moderation_state = 'flagged'
    WHEN @filter::text = 'rejected' THEN moderation_state = 'rejected'
    WHEN @filter::text = 'deleted' THEN moderation_state = 'deleted'
    WHEN @filter::text = 'private' THEN moderation_state NOT IN ('published', 'approved')
    ELSE true
  END
  AND (@search::text = ''
     OR title ILIKE '%' || @search::text || '%'
     OR name ILIKE '%' || @search::text || '%'
     OR id = @search::text);

-- name: GetSchematicByIDAdmin :one
SELECT * FROM schematics WHERE id = $1;

-- name: UpdateSchematicDetectedLanguage :exec
UPDATE schematics SET detected_language = $2 WHERE id = $1;

-- name: ListApprovedSchematicIDsAndCreated :many
SELECT id, created FROM schematics
WHERE moderation_state IN ('published', 'approved');

-- name: BatchGetSchematicCategories :many
SELECT sc.schematic_id, c.id, c.key, c.name
FROM schematics_categories sc
JOIN schematic_categories c ON c.id = sc.category_id
WHERE sc.schematic_id = ANY($1::text[]);

-- name: BatchGetSchematicTags :many
SELECT st.schematic_id, t.id, t.key, t.name
FROM schematics_tags st
JOIN schematic_tags t ON t.id = st.tag_id
WHERE st.schematic_id = ANY($1::text[]);

-- name: ListSchematicsByAuthorAll :many
-- Lists ALL schematics by an author regardless of moderation state (except deleted).
SELECT * FROM schematics
WHERE author_id = $1
  AND moderation_state != 'deleted'
ORDER BY created DESC
LIMIT $2 OFFSET $3;

-- name: CountSchematicsByAuthorAll :one
SELECT COUNT(*) FROM schematics
WHERE author_id = $1
  AND moderation_state != 'deleted';

