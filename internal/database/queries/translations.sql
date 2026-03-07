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
  AND s.moderated = true
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
