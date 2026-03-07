-- name: GetAchievementByKey :one
SELECT * FROM achievements WHERE key = $1;

-- name: ListAchievements :many
SELECT * FROM achievements ORDER BY key;

-- name: ListUserAchievements :many
SELECT a.* FROM achievements a
JOIN user_achievements ua ON ua.achievement_id = a.id
WHERE ua.user_id = $1
ORDER BY ua.created DESC;

-- name: AwardAchievement :one
INSERT INTO user_achievements (id, user_id, achievement_id)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, achievement_id) DO NOTHING
RETURNING *;

-- name: HasAchievement :one
SELECT EXISTS(
    SELECT 1 FROM user_achievements
    WHERE user_id = $1 AND achievement_id = $2
) AS has_achievement;

-- name: CreatePointLog :one
INSERT INTO point_log (id, user_id, points, reason, description, earned_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, reason) DO NOTHING
RETURNING *;

-- name: GetPointLog :many
SELECT * FROM point_log WHERE user_id = $1 ORDER BY earned_at DESC;

-- name: SumUserPoints :one
SELECT COALESCE(SUM(points), 0)::INTEGER AS total_points
FROM point_log WHERE user_id = $1;
