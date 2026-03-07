ALTER TABLE mod_metadata ADD COLUMN blocksitems_matched BOOLEAN NOT NULL DEFAULT false;
-- Reset last_fetched so all mods are re-enriched with the new BlocksItems step
UPDATE mod_metadata SET last_fetched = NULL WHERE manually_set = false;
