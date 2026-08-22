DROP TABLE IF EXISTS moderation_checklist_items;

ALTER TABLE schematics DROP COLUMN IF EXISTS moderation_resubmit_count;
ALTER TABLE schematics DROP COLUMN IF EXISTS removed_images;
ALTER TABLE schematics DROP COLUMN IF EXISTS held_images;
