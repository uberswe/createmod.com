ALTER TABLE schematics ADD COLUMN rotation_images TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE temp_upload_images ADD COLUMN category TEXT NOT NULL DEFAULT 'gallery';
