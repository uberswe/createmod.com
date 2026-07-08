-- Legacy schematic_ratings rows with rating=0 predate the API's 1-5
-- validation and poisoned averages below Google's declared worstRating.
-- Aggregate queries now filter them; this recomputes the denormalized
-- columns once for existing rows.
UPDATE schematics SET
    avg_rating = COALESCE(sub.avg_r, 0),
    rating_count = COALESCE(sub.cnt, 0)
FROM (
    SELECT schematic_id, AVG(rating)::REAL AS avg_r, COUNT(*)::INTEGER AS cnt
    FROM schematic_ratings
    WHERE deleted IS NULL AND rating BETWEEN 1 AND 5
    GROUP BY schematic_id
) sub
WHERE schematics.id = sub.schematic_id;

UPDATE schematics SET avg_rating = 0, rating_count = 0
WHERE (avg_rating <> 0 OR rating_count <> 0)
  AND id NOT IN (
    SELECT DISTINCT schematic_id FROM schematic_ratings
    WHERE deleted IS NULL AND rating BETWEEN 1 AND 5
  );
