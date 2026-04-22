ALTER TABLE schematic_ratings ADD COLUMN IF NOT EXISTS deleted TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_schematic_ratings_deleted ON schematic_ratings (deleted) WHERE deleted IS NULL;

ALTER TABLE guides ADD COLUMN IF NOT EXISTS deleted TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_guides_deleted ON guides (deleted) WHERE deleted IS NULL;
