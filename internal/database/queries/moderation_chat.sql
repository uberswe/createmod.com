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
