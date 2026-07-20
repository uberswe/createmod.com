-- name: UpsertSchematicFingerprint :exec
INSERT INTO schematic_fingerprints (schematic_id, fp, version, computed_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (schematic_id) DO UPDATE SET
    fp = EXCLUDED.fp,
    version = EXCLUDED.version,
    computed_at = now();

-- name: GetSchematicFingerprint :one
SELECT * FROM schematic_fingerprints WHERE schematic_id = $1;

-- name: ListSchematicsNeedingFingerprint :many
SELECT s.id FROM schematics s
LEFT JOIN schematic_fingerprints sf ON sf.schematic_id = s.id
WHERE (sf.schematic_id IS NULL OR sf.version < $1 OR COALESCE(s.modified, s.created) > sf.computed_at)
  AND s.deleted IS NULL
ORDER BY (sf.schematic_id IS NULL) DESC, s.created DESC
LIMIT $2;

-- name: ListAllFingerprints :many
SELECT sf.schematic_id, sf.fp FROM schematic_fingerprints sf
JOIN schematics s ON s.id = sf.schematic_id
WHERE s.deleted IS NULL AND s.moderation_state IN ('published', 'approved')
  AND sf.version = $1;

-- name: DeleteSchematicFingerprint :exec
DELETE FROM schematic_fingerprints WHERE schematic_id = $1;
