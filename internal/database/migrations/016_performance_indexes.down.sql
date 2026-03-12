-- Phase 2: Drop pre-computed columns and their indexes
DROP INDEX IF EXISTS idx_schematics_avg_rating;
DROP INDEX IF EXISTS idx_schematics_trending;

ALTER TABLE schematics DROP COLUMN IF EXISTS rating_count;
ALTER TABLE schematics DROP COLUMN IF EXISTS avg_rating;
ALTER TABLE schematics DROP COLUMN IF EXISTS trending_score;

-- Phase 1: Drop performance indexes
DROP INDEX IF EXISTS idx_collections_schematics_position;
DROP INDEX IF EXISTS idx_temp_uploads_uploaded_by_created;
DROP INDEX IF EXISTS idx_schematics_created_partial;
DROP INDEX IF EXISTS idx_schematics_scheduled;
DROP INDEX IF EXISTS idx_schematic_tags_public;
DROP INDEX IF EXISTS idx_comments_schematic_approved_created;
DROP INDEX IF EXISTS idx_schematics_views;
DROP INDEX IF EXISTS idx_schematic_views_type_created;
