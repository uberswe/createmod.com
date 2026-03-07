import { Page } from '@playwright/test';

// Known test user credentials.
// Must match what global-setup.ts seeds.
export const TEST_USER_EMAIL = 'e2e-test@createmod.com';
export const TEST_USER_PASSWORD = 'E2eTestPass123!';

/**
 * Resolve the PocketBase base URL.
 * In CI the app proxies PocketBase at the same origin; PB_URL is set when
 * PocketBase runs on a separate port (e.g. localhost:8090).
 */
export function pbURL(): string {
  return process.env.PB_URL || process.env.APP_BASE_URL || 'http://localhost:8090';
}

/**
 * Ensure the E2E test user exists.
 * Idempotent: tries to create the user; if it already exists (400) it
 * authenticates instead and returns the existing record id.
 *
 * Returns { id, token } for the user.
 */
export async function seedTestUser(baseURL?: string): Promise<{ id: string; token: string }> {
  const pb = pbURL();

  // Try creating the user first
  const createParams = new URLSearchParams();
  createParams.set('email', TEST_USER_EMAIL);
  createParams.set('password', TEST_USER_PASSWORD);
  createParams.set('passwordConfirm', TEST_USER_PASSWORD);
  createParams.set('username', 'e2etest');
  const createResp = await fetch(`${pb}/api/collections/users/records`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: createParams.toString(),
  });

  if (createResp.ok) {
    const data = await createResp.json();
    // Authenticate to get a token
    const authResult = await authenticateUser(pb);
    return { id: data.id, token: authResult.token };
  }

  // User likely already exists — authenticate
  return authenticateUser(pb);
}

/**
 * Authenticate the test user via PocketBase API and return { id, token }.
 */
async function authenticateUser(pb: string): Promise<{ id: string; token: string }> {
  const params = new URLSearchParams();
  params.set('identity', TEST_USER_EMAIL);
  params.set('password', TEST_USER_PASSWORD);
  const resp = await fetch(`${pb}/api/collections/users/auth-with-password`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: params.toString(),
  });

  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`Failed to authenticate test user: ${resp.status} ${text}`);
  }

  const data = await resp.json();
  return { id: data.record.id, token: data.token };
}

/**
 * Log in via the app's /login form POST so the browser context gets the
 * `create-mod-auth` cookie set by the PocketBase OnRecordAuthRequest hook.
 *
 * Use this for tests that need a fully authenticated browser session.
 */
export async function login(page: Page, baseURL?: string): Promise<void> {
  const url = baseURL ?? process.env.APP_BASE_URL ?? 'http://localhost:8080';

  // POST to the login endpoint which proxies to PocketBase and sets the cookie
  await page.goto(url + '/login');
  await page.fill('input[name="username"]', TEST_USER_EMAIL);
  await page.fill('input[name="password"]', TEST_USER_PASSWORD);
  await page.click('button[type="submit"]');

  // Wait for navigation after successful login (redirects to /)
  await page.waitForURL('**/');
}

/**
 * Log in by directly setting the auth cookie on the browser context.
 * Faster than going through the UI form — useful for tests that don't
 * need to exercise the login page itself.
 */
export async function loginViaCookie(page: Page, baseURL?: string): Promise<void> {
  const pb = pbURL();
  const appURL = baseURL ?? process.env.APP_BASE_URL ?? 'http://localhost:8080';
  const domain = new URL(appURL).hostname;

  const params = new URLSearchParams();
  params.set('identity', TEST_USER_EMAIL);
  params.set('password', TEST_USER_PASSWORD);
  const resp = await fetch(`${pb}/api/collections/users/auth-with-password`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: params.toString(),
  });

  if (!resp.ok) {
    throw new Error(`loginViaCookie: auth failed ${resp.status}`);
  }

  const data = await resp.json();

  await page.context().addCookies([
    {
      name: 'create-mod-auth',
      value: data.token,
      domain,
      path: '/',
      httpOnly: true,
    },
  ]);
}
