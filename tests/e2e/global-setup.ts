import { seedTestUser } from './helpers/auth';
import { superuserToken, seedTestSchematic, seedTestCollection } from './helpers/seed';

/**
 * Playwright global setup — runs once before any test file.
 *
 * Seeds the test user, a schematic, and a collection so the
 * previously-skipped E2E tests have the data they need.
 *
 * Requires PocketBase to be running (the CI workflow and docker-compose
 * both start it before Playwright executes).
 */
async function globalSetup() {
  console.log('[global-setup] Seeding E2E test data...');

  // 1. Create (or verify) test user
  const user = await seedTestUser();
  console.log(`[global-setup] Test user ready: ${user.id}`);

  // 2. Obtain superuser token so we can set fields like "moderated"
  let adminToken: string;
  try {
    adminToken = await superuserToken();
  } catch {
    // If no superuser is available (e.g. fresh PB with no admin),
    // fall back to the user's own token.  Some fields (moderated) may
    // not be settable, but the tests degrade gracefully.
    console.warn('[global-setup] Superuser auth failed; falling back to user token');
    adminToken = user.token;
  }

  // 3. Seed schematic
  const schematic = await seedTestSchematic(user.id, adminToken);
  console.log(`[global-setup] Test schematic ready: ${schematic.id} (${schematic.name})`);

  // 4. Seed collection containing that schematic
  const collection = await seedTestCollection(user.id, [schematic.id], adminToken);
  console.log(`[global-setup] Test collection ready: ${collection.id} (${collection.slug})`);

  console.log('[global-setup] Seeding complete.');
}

export default globalSetup;
