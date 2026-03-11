-- name: CreateUserWebhook :one
INSERT INTO user_webhooks (user_id, webhook_url_encrypted)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserWebhookByUserID :one
SELECT * FROM user_webhooks WHERE user_id = $1;

-- name: UpdateUserWebhookURL :exec
UPDATE user_webhooks
SET webhook_url_encrypted = $2,
    active = true,
    consecutive_failures = 0,
    last_failure_at = NULL,
    last_failure_message = '',
    updated = NOW()
WHERE user_id = $1;

-- name: DeleteUserWebhook :exec
DELETE FROM user_webhooks WHERE user_id = $1;

-- name: ListActiveUserWebhooks :many
SELECT id, user_id, webhook_url_encrypted FROM user_webhooks WHERE active = true;

-- name: IncrementWebhookFailure :exec
UPDATE user_webhooks
SET consecutive_failures = consecutive_failures + 1,
    last_failure_at = NOW(),
    last_failure_message = $2,
    active = CASE WHEN consecutive_failures + 1 >= 3 THEN false ELSE active END,
    updated = NOW()
WHERE id = $1;

-- name: ResetWebhookFailures :exec
UPDATE user_webhooks
SET consecutive_failures = 0,
    last_failure_at = NULL,
    last_failure_message = '',
    updated = NOW()
WHERE id = $1;
