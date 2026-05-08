-- name: GetModerationThreadByContent :one
SELECT * FROM moderation_threads
WHERE content_type = $1 AND content_id = $2
LIMIT 1;

-- name: CreateModerationThread :one
INSERT INTO moderation_threads (content_type, content_id)
VALUES ($1, $2)
RETURNING *;

-- name: ListModerationMessagesByThread :many
SELECT * FROM moderation_messages
WHERE thread_id = $1
ORDER BY created ASC;

-- name: CreateModerationMessage :one
INSERT INTO moderation_messages (thread_id, author_id, is_moderator, body)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CountUserMessagesSinceLastModerator :one
SELECT COUNT(*) FROM moderation_messages m
WHERE m.thread_id = $1
  AND m.created > COALESCE(
    (SELECT MAX(m2.created) FROM moderation_messages m2 WHERE m2.thread_id = $1 AND m2.is_moderator = TRUE),
    '1970-01-01'::timestamptz
  );

-- name: MarkModerationThreadResolved :exec
UPDATE moderation_threads SET resolved_at = NOW(), resolved_by = $2, updated = NOW()
WHERE id = $1;

-- name: MarkModerationThreadCreatorRead :exec
UPDATE moderation_threads SET creator_last_read = NOW(), updated = NOW()
WHERE id = $1;

-- name: UpdateModerationThreadLastMessage :exec
UPDATE moderation_threads SET last_message_at = NOW(), updated = NOW()
WHERE id = $1;

-- name: MarkModerationThreadCreatorNotified :exec
UPDATE moderation_threads SET creator_notified = true, updated = NOW()
WHERE id = $1;

-- name: ListUnreadThreadsByCreator :many
SELECT t.* FROM moderation_threads t
JOIN schematics s ON s.id = t.content_id AND t.content_type = 'schematic'
WHERE s.author_id = $1
  AND t.resolved_at IS NULL
  AND (t.creator_last_read IS NULL OR t.last_message_at > t.creator_last_read)
ORDER BY t.last_message_at DESC NULLS LAST;

-- name: ListModerationThreadsByModerator :many
SELECT t.* FROM moderation_threads t
WHERE t.resolved_at IS NULL
ORDER BY t.last_message_at DESC NULLS LAST
LIMIT $1 OFFSET $2;
