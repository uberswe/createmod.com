-- Remove the tags added in the up migration
DELETE FROM schematic_tags WHERE key IN (
    'compact', 'mega-build', 'starter', 'tutorial',
    'auto-farm', 'ore-processing', 'tree-farm', 'mob-farm',
    'food-production', 'cobblestone-gen', 'item-sorting', 'smeltery',
    'mechanical-press', 'deployer', 'crushing-wheel', 'windmill',
    'water-wheel', 'steam-engine', 'conveyor-belt', 'gearbox',
    'steampunk', 'medieval', 'industrial', 'modern', 'fantasy',
    'train-station', 'locomotive', 'rail-network',
    'create-additions', 'create-deco', 'create-steam-rails'
);

-- Remove the categories added in the up migration
DELETE FROM schematic_categories WHERE key IN (
    'trains', 'logistics', 'power-generation', 'processing',
    'decoration', 'redstone', 'vehicles', 'storage'
);
