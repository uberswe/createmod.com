-- name: CreateSchematicFile :one
INSERT INTO schematic_files (schematic_id, filename, original_name, size, mime_type)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, schematic_id, filename, original_name, size, mime_type, created, updated;

-- name: ListSchematicFilesBySchematicID :many
SELECT id, schematic_id, filename, original_name, size, mime_type, created, updated
FROM schematic_files
WHERE schematic_id = $1
ORDER BY created ASC;

-- name: DeleteSchematicFile :exec
DELETE FROM schematic_files WHERE id = $1;

-- name: DeleteSchematicFilesBySchematicID :exec
DELETE FROM schematic_files WHERE schematic_id = $1;
