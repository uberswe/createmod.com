-- name: CreateRedditLink :one
INSERT INTO schematic_reddit_links (schematic_id, reddit_url, subreddit, post_title)
VALUES ($1, $2, $3, $4)
ON CONFLICT (reddit_url) DO UPDATE SET
    post_title = EXCLUDED.post_title,
    updated = NOW()
RETURNING *;

-- name: GetRedditLinksBySchematic :many
SELECT * FROM schematic_reddit_links
WHERE schematic_id = $1
ORDER BY created ASC;

-- name: DeleteRedditLink :exec
DELETE FROM schematic_reddit_links WHERE id = $1 AND schematic_id = $2;

-- name: ListStaleRedditLinks :many
SELECT * FROM schematic_reddit_links
WHERE last_fetched IS NULL OR last_fetched < NOW() - INTERVAL '6 hours'
ORDER BY last_fetched ASC NULLS FIRST
LIMIT $1;

-- name: UpdateRedditLinkMetadata :exec
UPDATE schematic_reddit_links SET
    post_title = $2,
    upvotes = $3,
    comment_count = $4,
    thumbnail_url = $5,
    last_fetched = NOW(),
    updated = NOW()
WHERE id = $1;
