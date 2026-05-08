-- name: UpsertSocialLink :exec
INSERT INTO user_social_links (id, user_id, platform, url, username, verified)
VALUES (gen_random_uuid()::text, $1, $2, $3, $4, $5)
ON CONFLICT (user_id, platform) DO UPDATE
SET url = EXCLUDED.url, username = EXCLUDED.username, verified = EXCLUDED.verified, updated = NOW();

-- name: ListSocialLinksByUser :many
SELECT * FROM user_social_links WHERE user_id = $1 ORDER BY platform;

-- name: GetSocialLinkByUserAndPlatform :one
SELECT * FROM user_social_links WHERE user_id = $1 AND platform = $2;

-- name: DeleteSocialLink :exec
DELETE FROM user_social_links WHERE user_id = $1 AND platform = $2;

-- name: ListSocialLinksByPlatform :many
SELECT * FROM user_social_links WHERE platform = $1 ORDER BY created DESC;
