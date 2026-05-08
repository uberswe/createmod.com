-- name: CreateSchematicReference :one
INSERT INTO schematic_references (schematic_id, url, source_type, title, thumbnail_url, author_name)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (schematic_id, url) DO UPDATE SET
    source_type = EXCLUDED.source_type,
    title = EXCLUDED.title,
    thumbnail_url = EXCLUDED.thumbnail_url,
    author_name = EXCLUDED.author_name,
    updated = NOW()
RETURNING *;

-- name: ListSchematicReferences :many
SELECT * FROM schematic_references
WHERE schematic_id = $1
ORDER BY created ASC;

-- name: DeleteSchematicReference :exec
DELETE FROM schematic_references WHERE id = $1 AND schematic_id = $2;

-- name: ListStaleReferences :many
SELECT * FROM schematic_references
WHERE last_fetched IS NULL OR last_fetched < NOW() - INTERVAL '7 days'
ORDER BY last_fetched ASC NULLS FIRST
LIMIT $1;

-- name: UpdateReferenceMetadata :exec
UPDATE schematic_references SET
    title = $2,
    thumbnail_url = $3,
    author_name = $4,
    last_fetched = NOW(),
    updated = NOW()
WHERE id = $1;
