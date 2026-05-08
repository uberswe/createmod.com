-- name: GetBadgeByKey :one
SELECT * FROM badges WHERE key = $1;

-- name: ListBadges :many
SELECT * FROM badges ORDER BY category, title;

-- name: ListUserBadges :many
SELECT ub.id, ub.user_id, ub.badge_id, ub.count,
       b.key, b.title, b.description, b.icon, b.category, b.threshold, b.multi_earn
FROM user_badges ub
JOIN badges b ON b.id = ub.badge_id
WHERE ub.user_id = $1
ORDER BY ub.created DESC;

-- name: AwardBadge :exec
INSERT INTO user_badges (id, user_id, badge_id, count)
VALUES (gen_random_uuid()::text, $1, $2, 1)
ON CONFLICT (user_id, badge_id) DO NOTHING;

-- name: IncrementBadge :exec
INSERT INTO user_badges (id, user_id, badge_id, count)
VALUES (gen_random_uuid()::text, $1, $2, 1)
ON CONFLICT (user_id, badge_id) DO UPDATE SET count = user_badges.count + 1;

-- name: RemoveBadge :exec
DELETE FROM user_badges WHERE user_id = $1 AND badge_id = $2;

-- name: SetDisplayedBadge :exec
INSERT INTO user_displayed_badges (id, user_id, badge_id, position)
VALUES (gen_random_uuid()::text, $1, $2, $3)
ON CONFLICT (user_id, position) DO UPDATE SET badge_id = EXCLUDED.badge_id;

-- name: ClearDisplayedBadges :exec
DELETE FROM user_displayed_badges WHERE user_id = $1;

-- name: GetDisplayedBadges :many
SELECT udb.user_id, udb.badge_id, udb.position,
       b.key, b.title, b.description, b.icon, b.category, b.threshold, b.multi_earn
FROM user_displayed_badges udb
JOIN badges b ON b.id = udb.badge_id
WHERE udb.user_id = $1
ORDER BY udb.position;

-- name: BatchGetDisplayedBadges :many
SELECT udb.user_id, udb.badge_id, udb.position,
       b.key, b.title, b.description, b.icon, b.category, b.threshold, b.multi_earn
FROM user_displayed_badges udb
JOIN badges b ON b.id = udb.badge_id
WHERE udb.user_id = ANY($1::text[])
ORDER BY udb.user_id, udb.position;
