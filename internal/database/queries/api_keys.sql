-- name: GetAPIKeyByLast8 :one
SELECT * FROM api_keys WHERE last8 = $1;

-- name: ListAPIKeysByUser :many
SELECT * FROM api_keys WHERE user_id = $1 ORDER BY created DESC;

-- name: CreateAPIKey :one
INSERT INTO api_keys (id, user_id, key_hash, label, last8)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: DeleteAPIKey :exec
DELETE FROM api_keys WHERE id = $1 AND user_id = $2;

-- name: LogAPIKeyUsage :exec
INSERT INTO api_key_usage (id, api_key_id, endpoint)
VALUES ($1, $2, $3);

-- name: UpdateAPIKeyRateLimit :exec
UPDATE api_keys SET rate_limit_per_minute = $2, updated = NOW() WHERE id = $1;

-- name: ListAllAPIKeysWithUsage :many
SELECT k.id, k.user_id, k.label, k.last8, k.created, k.rate_limit_per_minute,
       COALESCE(u.username, '') AS username,
       COALESCE(us.total, 0)::BIGINT AS usage_total,
       COALESCE(us.last_24h, 0)::BIGINT AS usage_24h,
       COALESCE(us.last_7d, 0)::BIGINT AS usage_7d,
       COALESCE(us.last_used, '0001-01-01 00:00:00+00')::TIMESTAMPTZ AS last_used
FROM api_keys k
LEFT JOIN users u ON u.id = k.user_id
LEFT JOIN LATERAL (
    SELECT COUNT(*) AS total,
           COUNT(*) FILTER (WHERE created > NOW() - INTERVAL '24 hours') AS last_24h,
           COUNT(*) FILTER (WHERE created > NOW() - INTERVAL '7 days') AS last_7d,
           MAX(created) AS last_used
    FROM api_key_usage
    WHERE api_key_id = k.id
) us ON TRUE
ORDER BY k.created DESC;

-- name: ListAPIKeyUsageByEndpoint :many
SELECT api_key_id, endpoint, COUNT(*)::BIGINT AS requests, COALESCE(MAX(created), '0001-01-01 00:00:00+00')::TIMESTAMPTZ AS last_used
FROM api_key_usage
WHERE created > NOW() - INTERVAL '30 days'
GROUP BY api_key_id, endpoint
ORDER BY api_key_id, requests DESC;
