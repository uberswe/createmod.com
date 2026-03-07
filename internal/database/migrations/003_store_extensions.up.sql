-- Add unique constraint on schematic_views for upsert support
CREATE UNIQUE INDEX IF NOT EXISTS idx_schematic_views_unique
    ON schematic_views (schematic_id, type, period);

-- Add views column to guides for view tracking
ALTER TABLE guides ADD COLUMN IF NOT EXISTS views INTEGER NOT NULL DEFAULT 0;
