ALTER TABLE schematic_ratings ADD COLUMN deleted TIMESTAMPTZ;
CREATE INDEX idx_schematic_ratings_deleted ON schematic_ratings (deleted) WHERE deleted IS NULL;

ALTER TABLE guides ADD COLUMN deleted TIMESTAMPTZ;
CREATE INDEX idx_guides_deleted ON guides (deleted) WHERE deleted IS NULL;
