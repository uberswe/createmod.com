-- name: UpsertTrafficStat :exec
INSERT INTO traffic_stats (day, event_type, user_agent, country, resolution, page_class, count)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (day, event_type, user_agent, country, resolution, page_class)
DO UPDATE SET count = traffic_stats.count + excluded.count;

-- name: DeleteTrafficStatsBefore :exec
DELETE FROM traffic_stats WHERE day < $1;
