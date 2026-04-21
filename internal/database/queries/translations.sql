-- name: GetSchematicTranslation :one
SELECT * FROM schematic_translations
WHERE schematic_id = $1 AND language = $2;

-- name: ListSchematicTranslations :many
SELECT * FROM schematic_translations
WHERE schematic_id = $1
ORDER BY language;

-- name: UpsertSchematicTranslation :one
INSERT INTO schematic_translations (id, schematic_id, language, title, description, content)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (schematic_id, language) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    content = EXCLUDED.content
RETURNING *;

-- name: ListSchematicsWithoutTranslation :many
SELECT s.id, s.title, s.description, s.content
FROM schematics s
WHERE s.deleted IS NULL
  AND s.moderation_state = 'published'
  AND NOT EXISTS (
      SELECT 1 FROM schematic_translations st
      WHERE st.schematic_id = s.id AND st.language = $1
  )
ORDER BY s.created DESC
LIMIT $2;

-- name: GetGuideTranslation :one
SELECT * FROM guide_translations
WHERE guide_id = $1 AND language = $2;

-- name: ListGuideTranslations :many
SELECT * FROM guide_translations WHERE guide_id = $1 ORDER BY language;

-- name: UpsertGuideTranslation :one
INSERT INTO guide_translations (id, guide_id, language, title, description, content)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (guide_id, language) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    content = EXCLUDED.content
RETURNING *;

-- name: GetCommentTranslation :one
SELECT * FROM comment_translations
WHERE comment_id = $1 AND language = $2;

-- name: UpsertCommentTranslation :one
INSERT INTO comment_translations (id, comment_id, language, content)
VALUES ($1, $2, $3, $4)
ON CONFLICT (comment_id, language) DO UPDATE SET
    content = EXCLUDED.content,
    updated = NOW()
RETURNING *;

-- name: ListCommentsWithoutTranslation :many
SELECT c.id, c.author_id, c.schematic_id, c.parent_id, c.content,
       c.published, c.approved, c.type, c.karma, c.created, c.updated
FROM comments c
WHERE c.approved = true AND c.deleted IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM comment_translations ct
      WHERE ct.comment_id = c.id AND ct.language = $1
  )
ORDER BY c.created DESC
LIMIT $2;

-- name: GetCollectionTranslation :one
SELECT * FROM collection_translations
WHERE collection_id = $1 AND language = $2;

-- name: UpsertCollectionTranslation :one
INSERT INTO collection_translations (id, collection_id, language, title, description)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (collection_id, language) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description
RETURNING *;
