-- Preserve the originally uploaded file so it can be downloaded verbatim.
-- Uploads are normalized to Create .nbt for the rest of the pipeline; these
-- columns record the source format and where the untouched original lives so
-- non-.nbt uploads (e.g. Aeronautics .excraft, which has no writer) can be
-- offered back exactly as uploaded.
ALTER TABLE temp_uploads ADD COLUMN IF NOT EXISTS source_format TEXT NOT NULL DEFAULT '';
ALTER TABLE temp_uploads ADD COLUMN IF NOT EXISTS original_s3_key TEXT NOT NULL DEFAULT '';

ALTER TABLE schematics ADD COLUMN IF NOT EXISTS source_format TEXT NOT NULL DEFAULT '';
ALTER TABLE schematics ADD COLUMN IF NOT EXISTS original_file TEXT NOT NULL DEFAULT '';
