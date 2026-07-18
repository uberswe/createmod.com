-- name: CreateEditorSession :one
INSERT INTO editor_sessions (user_id, source_kind, source_ref)
VALUES ($1, $2, $3)
RETURNING id;

-- name: GetEditorSession :one
SELECT * FROM editor_sessions WHERE id = $1;

-- name: UpdateEditorSessionOps :exec
UPDATE editor_sessions SET ops = $2, cursor = $3, updated = now() WHERE id = $1;

-- name: DeleteExpiredEditorSessions :execrows
DELETE FROM editor_sessions WHERE updated < $1;
