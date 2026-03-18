-- name: CreateSchematicVariation :one
INSERT INTO schematic_variations (schematic_id, user_id, name, replacements, is_public)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetSchematicVariationByID :one
SELECT * FROM schematic_variations WHERE id = $1;

-- name: ListSchematicVariationsBySchematicAndUser :many
SELECT * FROM schematic_variations
WHERE schematic_id = $1 AND user_id = $2
ORDER BY created DESC;

-- name: ListPublicSchematicVariationsBySchematic :many
SELECT * FROM schematic_variations
WHERE schematic_id = $1 AND is_public = true
ORDER BY created DESC;

-- name: UpdateSchematicVariation :exec
UPDATE schematic_variations
SET name = $2, replacements = $3, is_public = $4, updated = NOW()
WHERE id = $1;

-- name: DeleteSchematicVariation :exec
DELETE FROM schematic_variations WHERE id = $1;

-- name: CountSchematicVariationsBySchematicAndUser :one
SELECT COUNT(*)::int FROM schematic_variations
WHERE schematic_id = $1 AND user_id = $2;

-- name: GetOldestSchematicVariationBySchematicAndUser :one
SELECT * FROM schematic_variations
WHERE schematic_id = $1 AND user_id = $2
ORDER BY created ASC
LIMIT 1;
