-- name: CreateSchematicVideo :one
INSERT INTO schematic_videos (schematic_id, video_url, video_type, title, position)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListSchematicVideos :many
SELECT * FROM schematic_videos
WHERE schematic_id = $1
ORDER BY position ASC, created ASC;

-- name: DeleteSchematicVideo :exec
DELETE FROM schematic_videos WHERE id = $1 AND schematic_id = $2;

-- name: DeleteSchematicVideosBySchematic :exec
DELETE FROM schematic_videos WHERE schematic_id = $1;

-- name: UpdateSchematicVideoPosition :exec
UPDATE schematic_videos SET position = $2 WHERE id = $1;

-- name: BatchGetSchematicVideos :many
SELECT * FROM schematic_videos
WHERE schematic_id = ANY($1::text[])
ORDER BY schematic_id, position ASC;
