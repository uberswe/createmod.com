-- name: CreateFollow :exec
INSERT INTO user_follows (user_id, follow_type, target_id, email_frequency, unsubscribe_token)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, follow_type, target_id) DO UPDATE SET
    email_frequency = EXCLUDED.email_frequency,
    unsubscribe_token = CASE WHEN EXCLUDED.unsubscribe_token != '' THEN EXCLUDED.unsubscribe_token ELSE user_follows.unsubscribe_token END;

-- name: DeleteFollow :exec
DELETE FROM user_follows WHERE user_id = $1 AND follow_type = $2 AND target_id = $3;

-- name: IsFollowing :one
SELECT EXISTS(
    SELECT 1 FROM user_follows WHERE user_id = $1 AND follow_type = $2 AND target_id = $3
) AS is_following;

-- name: GetFollow :one
SELECT * FROM user_follows
WHERE user_id = $1 AND follow_type = $2 AND target_id = $3
LIMIT 1;

-- name: UpdateFollowFrequency :exec
UPDATE user_follows SET email_frequency = $4
WHERE user_id = $1 AND follow_type = $2 AND target_id = $3;

-- name: UpdateFollowLastNotified :exec
UPDATE user_follows SET last_notified = NOW()
WHERE id = $1;

-- name: UnsubscribeFollow :exec
UPDATE user_follows SET email_frequency = 'off'
WHERE unsubscribe_token = $1 AND unsubscribe_token != '';

-- name: ListFollowsByUser :many
SELECT * FROM user_follows
WHERE user_id = $1
ORDER BY created DESC;

-- name: ListFollowsByUserAndType :many
SELECT * FROM user_follows
WHERE user_id = $1 AND follow_type = $2
ORDER BY created DESC;

-- name: ListFollowsByTarget :many
SELECT * FROM user_follows
WHERE follow_type = $1 AND target_id = $2
ORDER BY created ASC;

-- name: ListFollowsByFrequency :many
SELECT * FROM user_follows
WHERE email_frequency = $1
ORDER BY user_id, follow_type;

-- name: ListFollowedUserIDs :many
SELECT target_id FROM user_follows
WHERE user_id = $1 AND follow_type = 'user';

-- User-specific queries for profile follower counts
-- name: ListFollowerUsers :many
SELECT u.* FROM users u
JOIN user_follows uf ON uf.user_id = u.id
WHERE uf.follow_type = 'user' AND uf.target_id = $1
ORDER BY uf.created DESC
LIMIT $2 OFFSET $3;

-- name: ListFollowingUsers :many
SELECT u.* FROM users u
JOIN user_follows uf ON uf.target_id = u.id
WHERE uf.follow_type = 'user' AND uf.user_id = $1
ORDER BY uf.created DESC
LIMIT $2 OFFSET $3;

-- name: CountUserFollowers :one
SELECT COUNT(*)::INTEGER FROM user_follows
WHERE follow_type = 'user' AND target_id = $1;

-- name: CountUserFollowing :one
SELECT COUNT(*)::INTEGER FROM user_follows
WHERE follow_type = 'user' AND user_id = $1;

-- name: UpdateFollowerCountIncrement :exec
UPDATE users SET follower_count = follower_count + 1 WHERE id = $1;

-- name: UpdateFollowingCountIncrement :exec
UPDATE users SET following_count = following_count + 1 WHERE id = $1;

-- name: UpdateFollowerCountDecrement :exec
UPDATE users SET follower_count = GREATEST(follower_count - 1, 0) WHERE id = $1;

-- name: UpdateFollowingCountDecrement :exec
UPDATE users SET following_count = GREATEST(following_count - 1, 0) WHERE id = $1;
