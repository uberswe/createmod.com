DROP INDEX IF EXISTS idx_schematics_short_code;
ALTER TABLE schematics DROP COLUMN IF EXISTS short_code;
