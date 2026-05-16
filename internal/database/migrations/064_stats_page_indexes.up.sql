CREATE INDEX IF NOT EXISTS idx_schematic_downloads_created
    ON schematic_downloads(created);

CREATE INDEX IF NOT EXISTS idx_schematic_views_type5_period
    ON schematic_views(created, period, count) WHERE type = '5';

CREATE INDEX IF NOT EXISTS idx_schematic_views_type5_schematic
    ON schematic_views(schematic_id, count) WHERE type = '5';
