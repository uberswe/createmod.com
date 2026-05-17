-- Remove em dashes from Discord webhook guide
UPDATE guides SET content = replace(content, ' — ', ' - ')
WHERE id = 'guide_discord_webhook_001';

-- Fix Empty Schematic crafting recipe (Paper + Light Blue Dye, not just paper)
UPDATE guides SET content = replace(
    content,
    'Craft an <strong>Empty Schematic</strong> (paper in a crafting table)',
    'Craft an <strong>Empty Schematic</strong> (Paper + Light Blue Dye in a crafting table)'
) WHERE id = 'guide_install_001';
