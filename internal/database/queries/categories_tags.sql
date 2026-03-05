-- name: ListCategories :many
SELECT * FROM schematic_categories WHERE public = true ORDER BY key;

-- name: GetCategoryByID :one
SELECT * FROM schematic_categories WHERE id = $1;

-- name: GetCategoryByKey :one
SELECT * FROM schematic_categories WHERE key = $1 AND public = true;

-- name: GetCategoriesByIDs :many
SELECT * FROM schematic_categories WHERE id = ANY($1::text[]);

-- name: ListAllCategories :many
SELECT * FROM schematic_categories ORDER BY key;

-- name: ListPendingCategories :many
SELECT * FROM schematic_categories WHERE public = false ORDER BY created DESC;

-- name: ApproveCategoryByID :exec
UPDATE schematic_categories SET public = true WHERE id = $1;

-- name: DeleteCategoryByID :exec
DELETE FROM schematic_categories WHERE id = $1;

-- name: GetCategoryByKeyIncludingPending :one
SELECT * FROM schematic_categories WHERE key = $1;

-- name: ListTags :many
SELECT * FROM schematic_tags WHERE public = true ORDER BY key;

-- name: GetTagByID :one
SELECT * FROM schematic_tags WHERE id = $1;

-- name: GetTagByKey :one
SELECT * FROM schematic_tags WHERE key = $1 AND public = true;

-- name: GetTagsByIDs :many
SELECT * FROM schematic_tags WHERE id = ANY($1::text[]);

-- name: ListTagsWithCount :many
SELECT t.id, t.key, t.name, COUNT(st.schematic_id) AS count
FROM schematic_tags t
LEFT JOIN schematics_tags st ON st.tag_id = t.id
LEFT JOIN schematics s ON s.id = st.schematic_id
    AND s.deleted IS NULL AND s.moderated = true
WHERE t.public = true
GROUP BY t.id, t.key, t.name
ORDER BY count DESC;

-- name: ListAllTags :many
SELECT * FROM schematic_tags ORDER BY key;

-- name: ListPendingTags :many
SELECT * FROM schematic_tags WHERE public = false ORDER BY created DESC;

-- name: ApproveTagByID :exec
UPDATE schematic_tags SET public = true WHERE id = $1;

-- name: DeleteTagByID :exec
DELETE FROM schematic_tags WHERE id = $1;

-- name: GetTagByKeyIncludingPending :one
SELECT * FROM schematic_tags WHERE key = $1;

-- name: CreateCategory :one
INSERT INTO schematic_categories (id, key, name, public)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CreateTag :one
INSERT INTO schematic_tags (id, key, name, public)
VALUES ($1, $2, $3, $4)
RETURNING *;
