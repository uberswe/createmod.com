-- name: UpsertSchematicSafety :exec
INSERT INTO schematic_safety (schematic_id, checksum, file_safe, manifest, pipeline_version, scanned_at)
VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT (schematic_id) DO UPDATE SET
    checksum = EXCLUDED.checksum,
    file_safe = EXCLUDED.file_safe,
    manifest = EXCLUDED.manifest,
    pipeline_version = EXCLUDED.pipeline_version,
    scanned_at = now();

-- name: GetSchematicSafety :one
SELECT * FROM schematic_safety WHERE schematic_id = $1;

-- name: ListSchematicsNeedingSafetyScan :many
SELECT s.id FROM schematics s
LEFT JOIN schematic_safety ss ON ss.schematic_id = s.id
WHERE (ss.schematic_id IS NULL OR ss.pipeline_version < $1 OR COALESCE(s.modified, s.created) > ss.scanned_at)
  AND s.deleted IS NULL
ORDER BY (ss.schematic_id IS NULL) DESC, s.created DESC
LIMIT $2;

-- name: DeleteSchematicSafety :exec
DELETE FROM schematic_safety WHERE schematic_id = $1;
