-- Add new categories based on common Create mod build types
-- that are frequently searched for but don't have a dedicated category.

INSERT INTO schematic_categories (id, key, name, created, updated)
VALUES
    (gen_random_uuid()::TEXT, 'trains', 'Trains & Railways', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'logistics', 'Logistics & Item Transport', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'power-generation', 'Power Generation', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'processing', 'Processing & Automation', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'decoration', 'Decoration & Aesthetics', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'redstone', 'Redstone & Logic', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'vehicles', 'Vehicles & Contraptions', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'storage', 'Storage Systems', NOW(), NOW())
ON CONFLICT (key) DO NOTHING;

-- Mark new categories as public
UPDATE schematic_categories SET public = true
WHERE key IN ('trains', 'logistics', 'power-generation', 'processing', 'decoration', 'redstone', 'vehicles', 'storage');

-- Add new tags for commonly searched terms
INSERT INTO schematic_tags (id, key, name, created, updated)
VALUES
    -- Build scale/complexity tags
    (gen_random_uuid()::TEXT, 'compact', 'Compact', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'mega-build', 'Mega Build', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'starter', 'Starter Friendly', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'tutorial', 'Tutorial Build', NOW(), NOW()),
    -- Functional tags
    (gen_random_uuid()::TEXT, 'auto-farm', 'Auto Farm', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'ore-processing', 'Ore Processing', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'tree-farm', 'Tree Farm', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'mob-farm', 'Mob Farm', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'food-production', 'Food Production', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'cobblestone-gen', 'Cobblestone Generator', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'item-sorting', 'Item Sorting', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'smeltery', 'Smeltery', NOW(), NOW()),
    -- Create-specific mechanism tags
    (gen_random_uuid()::TEXT, 'mechanical-press', 'Mechanical Press', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'deployer', 'Deployer', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'crushing-wheel', 'Crushing Wheel', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'windmill', 'Windmill', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'water-wheel', 'Water Wheel', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'steam-engine', 'Steam Engine', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'conveyor-belt', 'Conveyor Belt', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'gearbox', 'Gearbox', NOW(), NOW()),
    -- Aesthetic/style tags
    (gen_random_uuid()::TEXT, 'steampunk', 'Steampunk', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'medieval', 'Medieval', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'industrial', 'Industrial', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'modern', 'Modern', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'fantasy', 'Fantasy', NOW(), NOW()),
    -- Train-related tags
    (gen_random_uuid()::TEXT, 'train-station', 'Train Station', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'locomotive', 'Locomotive', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'rail-network', 'Rail Network', NOW(), NOW()),
    -- Addon/integration tags
    (gen_random_uuid()::TEXT, 'create-additions', 'Create: Additions', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'create-deco', 'Create: Deco', NOW(), NOW()),
    (gen_random_uuid()::TEXT, 'create-steam-rails', 'Create: Steam n Rails', NOW(), NOW())
ON CONFLICT (key) DO NOTHING;

-- Mark new tags as public
UPDATE schematic_tags SET public = true
WHERE key IN (
    'compact', 'mega-build', 'starter', 'tutorial',
    'auto-farm', 'ore-processing', 'tree-farm', 'mob-farm',
    'food-production', 'cobblestone-gen', 'item-sorting', 'smeltery',
    'mechanical-press', 'deployer', 'crushing-wheel', 'windmill',
    'water-wheel', 'steam-engine', 'conveyor-belt', 'gearbox',
    'steampunk', 'medieval', 'industrial', 'modern', 'fantasy',
    'train-station', 'locomotive', 'rail-network',
    'create-additions', 'create-deco', 'create-steam-rails'
);
