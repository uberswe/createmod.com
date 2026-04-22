DROP INDEX IF EXISTS idx_guides_deleted;
ALTER TABLE guides DROP COLUMN IF EXISTS deleted;

DROP INDEX IF EXISTS idx_schematic_ratings_deleted;
ALTER TABLE schematic_ratings DROP COLUMN IF EXISTS deleted;
