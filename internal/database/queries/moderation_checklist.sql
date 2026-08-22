-- name: CreateModerationChecklistItem :one
INSERT INTO moderation_checklist_items (schematic_id, kind, source, note, status)
VALUES ($1, $2, $3, $4, COALESCE(sqlc.narg('status'), 'open'))
RETURNING *;

-- name: ListModerationChecklistBySchematic :many
SELECT * FROM moderation_checklist_items
WHERE schematic_id = $1
ORDER BY created_at ASC;

-- name: ListOpenModerationChecklistBySchematic :many
SELECT * FROM moderation_checklist_items
WHERE schematic_id = $1 AND status = 'open'
ORDER BY created_at ASC;

-- name: CountOpenModerationChecklistBySchematic :one
SELECT COUNT(*) FROM moderation_checklist_items
WHERE schematic_id = $1 AND status = 'open';

-- name: ResolveModerationChecklistItem :exec
UPDATE moderation_checklist_items
SET status = 'resolved', resolved_at = NOW()
WHERE id = $1 AND status = 'open';

-- name: ResolveOpenModerationChecklistByKind :execrows
UPDATE moderation_checklist_items
SET status = 'resolved', resolved_at = NOW()
WHERE schematic_id = $1 AND kind = $2 AND status = 'open';

-- name: DeleteModerationChecklistBySchematic :exec
DELETE FROM moderation_checklist_items WHERE schematic_id = $1;
