-- Unpublish collections that have fewer than 4 schematics.
UPDATE collections
SET published = false
WHERE published = true
  AND deleted = ''
  AND (
    SELECT COUNT(*)
    FROM collections_schematics cs
    WHERE cs.collection_id = collections.id
  ) < 4;
