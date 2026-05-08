-- name: UpsertZeroResultSuggestion :one
INSERT INTO zero_result_suggestions (query, suggestion, auto)
VALUES ($1, $2, $3)
ON CONFLICT (query) DO UPDATE SET
    suggestion = EXCLUDED.suggestion,
    auto = EXCLUDED.auto,
    updated = NOW()
RETURNING *;

-- name: GetZeroResultSuggestion :one
SELECT * FROM zero_result_suggestions
WHERE query = $1
LIMIT 1;

-- name: ListZeroResultSuggestions :many
SELECT * FROM zero_result_suggestions
ORDER BY updated DESC
LIMIT $1 OFFSET $2;

-- name: DeleteZeroResultSuggestion :exec
DELETE FROM zero_result_suggestions WHERE id = $1;
