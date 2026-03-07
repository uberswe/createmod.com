import { seedTestUser } from './helpers/auth';

/**
 * Playwright global setup — runs once before any test file.
 *
 * In CI, test data (user, schematic, collection) is seeded via seed.sql
 * before the app starts. This setup verifies the app is healthy and
 * ensures the test user can authenticate.
 */
async function globalSetup() {
  const url = process.env.APP_BASE_URL || 'http://localhost:8080';

  // Verify app is running
  console.log('[global-setup] Checking app health...');
  const resp = await fetch(`${url}/api/health`);
  if (!resp.ok) {
    throw new Error(`App health check failed: ${resp.status}`);
  }
  console.log('[global-setup] App is healthy.');

  // Ensure test user exists (registers via /register as fallback for local dev)
  console.log('[global-setup] Ensuring test user exists...');
  await seedTestUser();
  console.log('[global-setup] Test user ready.');
}

export default globalSetup;
