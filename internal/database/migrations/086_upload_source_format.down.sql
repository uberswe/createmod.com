ALTER TABLE schematics DROP COLUMN IF EXISTS original_file;
ALTER TABLE schematics DROP COLUMN IF EXISTS source_format;

ALTER TABLE temp_uploads DROP COLUMN IF EXISTS original_s3_key;
ALTER TABLE temp_uploads DROP COLUMN IF EXISTS source_format;
