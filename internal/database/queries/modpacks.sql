-- name: UpsertModpack :one
INSERT INTO modpacks (modrinth_id, slug, name, description, icon_url, modrinth_url, downloads, last_fetched)
VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
ON CONFLICT (modrinth_id) DO UPDATE SET
    slug = EXCLUDED.slug,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    icon_url = EXCLUDED.icon_url,
    modrinth_url = EXCLUDED.modrinth_url,
    downloads = EXCLUDED.downloads,
    last_fetched = NOW(),
    updated = NOW()
RETURNING *;

-- name: GetModpackByID :one
SELECT * FROM modpacks WHERE id = $1 LIMIT 1;

-- name: GetModpackBySlug :one
SELECT * FROM modpacks WHERE slug = $1 LIMIT 1;

-- name: ListModpacks :many
SELECT * FROM modpacks ORDER BY name ASC;

-- name: SearchModpacks :many
SELECT * FROM modpacks
WHERE name ILIKE '%' || $1 || '%' OR slug ILIKE '%' || $1 || '%'
ORDER BY downloads DESC
LIMIT $2;

-- name: SetSchematicModpacks :exec
DELETE FROM schematics_modpacks WHERE schematic_id = $1;

-- name: AddSchematicModpack :exec
INSERT INTO schematics_modpacks (schematic_id, modpack_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: GetSchematicModpacks :many
SELECT m.* FROM modpacks m
JOIN schematics_modpacks sm ON sm.modpack_id = m.id
WHERE sm.schematic_id = $1
ORDER BY m.name ASC;

-- name: BatchGetSchematicModpacks :many
SELECT sm.schematic_id, m.* FROM modpacks m
JOIN schematics_modpacks sm ON sm.modpack_id = m.id
WHERE sm.schematic_id = ANY($1::text[])
ORDER BY m.name ASC;

-- name: ListSchematicsByModpack :many
SELECT s.* FROM schematics s
JOIN schematics_modpacks sm ON sm.schematic_id = s.id
WHERE sm.modpack_id = $1
  AND s.moderation_state IN ('published', 'approved')
  AND s.deleted IS NULL
ORDER BY s.created DESC
LIMIT $2 OFFSET $3;

-- name: CountSchematicsByModpack :one
SELECT COUNT(*) FROM schematics s
JOIN schematics_modpacks sm ON sm.schematic_id = s.id
WHERE sm.modpack_id = $1
  AND s.moderation_state IN ('published', 'approved')
  AND s.deleted IS NULL;
