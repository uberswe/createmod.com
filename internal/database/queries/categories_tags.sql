-- name: ListCategories :many
SELECT * FROM schematic_categories ORDER BY key;

-- name: GetCategoryByID :one
SELECT * FROM schematic_categories WHERE id = $1;

-- name: GetCategoryByKey :one
SELECT * FROM schematic_categories WHERE key = $1;

-- name: GetCategoriesByIDs :many
SELECT * FROM schematic_categories WHERE id = ANY($1::text[]);

-- name: ListTags :many
SELECT * FROM schematic_tags ORDER BY key;

-- name: GetTagByID :one
SELECT * FROM schematic_tags WHERE id = $1;

-- name: GetTagByKey :one
SELECT * FROM schematic_tags WHERE key = $1;

-- name: GetTagsByIDs :many
SELECT * FROM schematic_tags WHERE id = ANY($1::text[]);

-- name: ListTagsWithCount :many
SELECT t.id, t.key, t.name, COUNT(st.schematic_id) AS count
FROM schematic_tags t
LEFT JOIN schematics_tags st ON st.tag_id = t.id
LEFT JOIN schematics s ON s.id = st.schematic_id
    AND s.deleted IS NULL AND s.moderated = true
GROUP BY t.id, t.key, t.name
ORDER BY count DESC;

-- name: CreateCategory :one
INSERT INTO schematic_categories (id, key, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateTag :one
INSERT INTO schematic_tags (id, key, name)
VALUES ($1, $2, $3)
RETURNING *;
