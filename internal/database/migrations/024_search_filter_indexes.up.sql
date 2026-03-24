CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_schematics_block_count ON schematics (block_count) WHERE deleted IS NULL AND moderated = true;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_schematics_dim_x ON schematics (dim_x) WHERE deleted IS NULL AND moderated = true;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_schematics_dim_y ON schematics (dim_y) WHERE deleted IS NULL AND moderated = true;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_schematics_dim_z ON schematics (dim_z) WHERE deleted IS NULL AND moderated = true;
