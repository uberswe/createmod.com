-- name: CreateModerationLog :one
INSERT INTO moderation_log (schematic_id, actor_id, actor_type, action, old_state, new_state, reason)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListAutoApprovedSchematicsSince :many
SELECT DISTINCT ON (ml.schematic_id) ml.schematic_id, ml.created_at, s.title, s.name
FROM moderation_log ml
JOIN schematics s ON s.id = ml.schematic_id
WHERE ml.actor_type = 'system'
  AND ml.new_state = 'published'
  AND ml.created_at >= @since AND ml.created_at < @until
  AND s.deleted IS NULL
ORDER BY ml.schematic_id, ml.created_at DESC;

-- name: ListModerationLogBySchematic :many
SELECT ml.*, COALESCE(u.username, '') AS actor_username
FROM moderation_log ml
LEFT JOIN users u ON ml.actor_id = u.id
WHERE ml.schematic_id = $1
ORDER BY ml.created_at DESC
LIMIT 50;
