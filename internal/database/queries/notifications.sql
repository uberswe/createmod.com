-- name: CreateNotification :one
INSERT INTO notifications (user_id, type, title, body, url, actor_id, reference_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListNotificationsByUser :many
SELECT * FROM notifications
WHERE user_id = $1
ORDER BY created DESC
LIMIT $2 OFFSET $3;

-- name: ListRecentNotifications :many
SELECT * FROM notifications
WHERE user_id = $1
ORDER BY created DESC
LIMIT $2;

-- name: CountUnreadNotifications :one
SELECT COUNT(*) FROM notifications
WHERE user_id = $1 AND read = false;

-- name: MarkNotificationRead :exec
UPDATE notifications SET read = true WHERE id = $1 AND user_id = $2;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications SET read = true WHERE user_id = $1 AND read = false;

-- name: DeleteOldNotifications :exec
DELETE FROM notifications WHERE read = true AND created < $1;

-- name: GetNotificationPreferences :many
SELECT * FROM notification_preferences
WHERE user_id = $1
ORDER BY category ASC;

-- name: UpsertNotificationPreference :one
INSERT INTO notification_preferences (user_id, category, email, web)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, category) DO UPDATE SET
    email = EXCLUDED.email,
    web = EXCLUDED.web,
    updated = NOW()
RETURNING *;

-- name: GetNotificationPreference :one
SELECT * FROM notification_preferences
WHERE user_id = $1 AND category = $2
LIMIT 1;
