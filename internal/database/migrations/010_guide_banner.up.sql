ALTER TABLE guides ADD COLUMN IF NOT EXISTS banner_url TEXT NOT NULL DEFAULT '';

-- Set default banners for seeded guides
UPDATE guides SET banner_url = '/assets/x/guides/schematic_upload.webp' WHERE id = 'guide_upload_001' AND banner_url = '';
UPDATE guides SET banner_url = '/assets/x/guides/schematic_download.webp' WHERE id = 'guide_install_001' AND banner_url = '';
UPDATE guides SET banner_url = '/assets/x/guides/createmod_get_started.webp' WHERE id = 'guide_getting_001' AND banner_url = '';
