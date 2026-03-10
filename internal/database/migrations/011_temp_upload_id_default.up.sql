-- Add default UUID generation for temp_uploads.id so INSERTs without
-- explicit id values work (matches temp_upload_files pattern).
ALTER TABLE temp_uploads ALTER COLUMN id SET DEFAULT gen_random_uuid()::text;
