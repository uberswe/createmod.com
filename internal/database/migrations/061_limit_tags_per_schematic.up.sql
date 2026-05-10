-- Remove excess tags from schematics that have more than 5.
-- Keeps 5 arbitrary tags per schematic (ordered by tag_id).
DELETE FROM schematics_tags
WHERE (schematic_id, tag_id) IN (
    SELECT schematic_id, tag_id
    FROM (
        SELECT schematic_id, tag_id,
               ROW_NUMBER() OVER (PARTITION BY schematic_id ORDER BY tag_id) AS rn
        FROM schematics_tags
    ) ranked
    WHERE rn > 5
);
