-- FetchTotalViewsBySchematic: partial covering index for the aggregation query
-- WHERE type = '0' GROUP BY schematic_id, SUM(count) -> index-only scan
CREATE INDEX IF NOT EXISTS idx_schematic_views_type0_agg
    ON schematic_views(schematic_id, count) WHERE type = '0';

-- FetchRecentViewsBySchematic: same pattern but includes created for the date filter
CREATE INDEX IF NOT EXISTS idx_schematic_views_type0_recent
    ON schematic_views(created, schematic_id, count) WHERE type = '0';

-- ListSchematicsByCategoryIDs: index leading with category_id for the ANY() lookup
-- (existing PK is (schematic_id, category_id) - wrong column order for this query)
CREATE INDEX IF NOT EXISTS idx_schematics_categories_cat_schematic
    ON schematics_categories(category_id, schematic_id);

-- ListTopSearches: hash index on query for GROUP BY aggregation (15s full table scan -> hash agg)
-- Using HASH because some query values exceed btree's 2704-byte row size limit.
-- Hash indexes support equality and GROUP BY which is all ListTopSearches needs.
CREATE INDEX IF NOT EXISTS idx_searches_query
    ON searches USING hash (query);
