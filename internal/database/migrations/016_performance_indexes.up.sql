-- Phase 1: Missing database indexes for common query patterns

CREATE INDEX IF NOT EXISTS idx_schematic_views_type_created ON schematic_views(type, created);
CREATE INDEX IF NOT EXISTS idx_schematics_views ON schematics(views) WHERE deleted IS NULL AND moderated = true;
CREATE INDEX IF NOT EXISTS idx_comments_schematic_approved_created ON comments(schematic_id, approved, created);
CREATE INDEX IF NOT EXISTS idx_schematic_tags_public ON schematic_tags(public);
CREATE INDEX IF NOT EXISTS idx_schematics_scheduled ON schematics(scheduled_at) WHERE scheduled_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_schematics_created_partial ON schematics(created) WHERE deleted IS NULL;
CREATE INDEX IF NOT EXISTS idx_temp_uploads_uploaded_by_created ON temp_uploads(uploaded_by, created);
CREATE INDEX IF NOT EXISTS idx_collections_schematics_position ON collections_schematics(collection_id, position);

-- Phase 2: Pre-computed rating aggregates and trending scores

ALTER TABLE schematics ADD COLUMN IF NOT EXISTS trending_score REAL NOT NULL DEFAULT 0;
ALTER TABLE schematics ADD COLUMN IF NOT EXISTS avg_rating REAL NOT NULL DEFAULT 0;
ALTER TABLE schematics ADD COLUMN IF NOT EXISTS rating_count INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_schematics_trending ON schematics(trending_score DESC) WHERE deleted IS NULL AND moderated = true;
CREATE INDEX IF NOT EXISTS idx_schematics_avg_rating ON schematics(avg_rating DESC, rating_count DESC) WHERE deleted IS NULL AND moderated = true;

-- Backfill from existing data so ListHighestRated works immediately
UPDATE schematics s SET avg_rating = sub.avg_r, rating_count = sub.cnt
FROM (SELECT schematic_id, AVG(rating)::REAL AS avg_r, COUNT(*)::INTEGER AS cnt FROM schematic_ratings GROUP BY schematic_id) sub
WHERE s.id = sub.schematic_id;
