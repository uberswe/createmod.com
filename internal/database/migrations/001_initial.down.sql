-- Reverse of 001_initial.up.sql
-- Drop in reverse dependency order

-- Drop triggers first
DO $$
DECLARE
    tbl TEXT;
BEGIN
    FOR tbl IN
        SELECT table_name
        FROM information_schema.columns
        WHERE column_name = 'updated'
          AND table_schema = 'public'
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS trg_%I_updated ON %I', tbl, tbl);
    END LOOP;
END;
$$;

DROP FUNCTION IF EXISTS update_updated_column();

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS outgoing_clicks;
DROP TABLE IF EXISTS contact_form_submissions;
DROP TABLE IF EXISTS pages;
DROP TABLE IF EXISTS news;
DROP TABLE IF EXISTS searches;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS temp_upload_files;
DROP TABLE IF EXISTS temp_uploads;
DROP TABLE IF EXISTS api_key_usage;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS point_log;
DROP TABLE IF EXISTS user_achievements;
DROP TABLE IF EXISTS achievements;
DROP TABLE IF EXISTS collection_translations;
DROP TABLE IF EXISTS collections_schematics;
DROP TABLE IF EXISTS collections;
DROP TABLE IF EXISTS guide_translations;
DROP TABLE IF EXISTS guides;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS nbt_hashes;
DROP TABLE IF EXISTS schematic_translations;
DROP TABLE IF EXISTS schematic_files;
DROP TABLE IF EXISTS schematic_versions;
DROP TABLE IF EXISTS schematic_downloads;
DROP TABLE IF EXISTS schematic_ratings;
DROP TABLE IF EXISTS schematic_views;
DROP TABLE IF EXISTS schematics_tags;
DROP TABLE IF EXISTS schematics_categories;
DROP TABLE IF EXISTS schematics;
DROP TABLE IF EXISTS minecraft_versions;
DROP TABLE IF EXISTS createmod_versions;
DROP TABLE IF EXISTS schematic_tags;
DROP TABLE IF EXISTS schematic_categories;
DROP TABLE IF EXISTS user_meta;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS external_auths;
DROP TABLE IF EXISTS mod_metadata;
DROP TABLE IF EXISTS users;
