-- name: CreateFollow :exec
WITH ins AS (
    INSERT INTO user_follows (id, follower_id, followed_id)
    VALUES (gen_random_uuid()::text, $1, $2)
    ON CONFLICT (follower_id, followed_id) DO NOTHING
    RETURNING follower_id, followed_id
)
SELECT * FROM ins;

-- name: UpdateFollowerCountIncrement :exec
UPDATE users SET follower_count = follower_count + 1 WHERE id = $1;

-- name: UpdateFollowingCountIncrement :exec
UPDATE users SET following_count = following_count + 1 WHERE id = $1;

-- name: DeleteFollow :exec
DELETE FROM user_follows WHERE follower_id = $1 AND followed_id = $2;

-- name: UpdateFollowerCountDecrement :exec
UPDATE users SET follower_count = GREATEST(follower_count - 1, 0) WHERE id = $1;

-- name: UpdateFollowingCountDecrement :exec
UPDATE users SET following_count = GREATEST(following_count - 1, 0) WHERE id = $1;

-- name: IsFollowing :one
SELECT EXISTS(
    SELECT 1 FROM user_follows WHERE follower_id = $1 AND followed_id = $2
) AS is_following;

-- name: ListFollowers :many
SELECT u.* FROM users u
JOIN user_follows uf ON uf.follower_id = u.id
WHERE uf.followed_id = $1
ORDER BY uf.created DESC
LIMIT $2 OFFSET $3;

-- name: ListFollowing :many
SELECT u.* FROM users u
JOIN user_follows uf ON uf.followed_id = u.id
WHERE uf.follower_id = $1
ORDER BY uf.created DESC
LIMIT $2 OFFSET $3;

-- name: CountFollowers :one
SELECT COUNT(*)::INTEGER FROM user_follows WHERE followed_id = $1;

-- name: CountFollowing :one
SELECT COUNT(*)::INTEGER FROM user_follows WHERE follower_id = $1;

-- name: ListFollowedIDs :many
SELECT followed_id FROM user_follows WHERE follower_id = $1;
