-- E2E test seed data.
-- Run after migrations: psql $DATABASE_URL -f tests/e2e/seed.sql
-- All INSERTs use ON CONFLICT DO NOTHING for idempotency.

-- 1. Test user: e2e-test@createmod.com / E2eTestPass123!
--    ID is a fixed 15-char hex string matching the app's generateID() format.
--    password_hash is bcrypt cost-12 of "E2eTestPass123!"
INSERT INTO users (id, email, username, password_hash, verified, is_admin, created, updated)
VALUES (
    'e2e000000000001',
    'e2e-test@createmod.com',
    'e2etest',
    '$2a$12$4SoNxwHo.ojjkVowt3EC7uQXAZRBetWm5pctKVL3lQjlNtBysvMWm',
    true,
    true,
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- 2. Test schematic: e2e-test-schematic (moderated, visible)
INSERT INTO schematics (id, author_id, name, title, description, moderated, featured_image, created, updated)
VALUES (
    'e2e000000000002',
    'e2e000000000001',
    'e2e-test-schematic',
    'E2E Test Schematic',
    'Schematic created by E2E seed SQL.',
    true,
    'test_featured.webp',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- 3. Test collection: e2e-test-collection (published, visible)
INSERT INTO collections (id, author_id, title, name, slug, description, published, created, updated)
VALUES (
    'e2e000000000003',
    'e2e000000000001',
    'E2E Test Collection',
    'E2E Test Collection',
    'e2e-test-collection',
    'Collection created by E2E seed SQL.',
    true,
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- 4. Link schematic to collection
INSERT INTO collections_schematics (collection_id, schematic_id, position)
VALUES ('e2e000000000003', 'e2e000000000002', 0)
ON CONFLICT (collection_id, schematic_id) DO NOTHING;
