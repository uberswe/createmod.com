import { Page } from '@playwright/test';

// Known test user credentials.
// Must match what seed.sql inserts.
export const TEST_USER_EMAIL = 'e2e-test@createmod.com';
export const TEST_USER_PASSWORD = 'E2eTestPass123!';
export const TEST_USER_USERNAME = 'e2etest';

/**
 * Resolve the app base URL.
 */
export function appURL(): string {
  return process.env.APP_BASE_URL || 'http://localhost:8080';
}

/**
 * Ensure the E2E test user exists.
 *
 * In CI the user is seeded via seed.sql before the app starts.
 * This function attempts to register the user via the app's /register
 * endpoint as a fallback (for local dev), then logs in to verify.
 *
 * Returns the session cookie value.
 */
export async function seedTestUser(): Promise<string> {
  const url = appURL();

  // Try registering — will fail gracefully if user already exists
  const form = new URLSearchParams();
  form.set('username', TEST_USER_USERNAME);
  form.set('email', TEST_USER_EMAIL);
  form.set('password', TEST_USER_PASSWORD);
  form.set('terms', 'on');

  await fetch(`${url}/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: form.toString(),
    redirect: 'manual', // don't follow redirects
  });

  // Verify we can authenticate
  const cookie = await authenticateUser();
  return cookie;
}

/**
 * Authenticate the test user via the app's /login endpoint.
 * Returns the create-mod-auth cookie value.
 */
export async function authenticateUser(): Promise<string> {
  const url = appURL();

  const form = new URLSearchParams();
  form.set('username', TEST_USER_EMAIL);
  form.set('password', TEST_USER_PASSWORD);

  const resp = await fetch(`${url}/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: form.toString(),
    redirect: 'manual', // don't follow — we need Set-Cookie from the 302
  });

  // Extract create-mod-auth from Set-Cookie header
  const setCookie = resp.headers.getSetCookie?.() ?? [];
  let token = '';
  for (const c of setCookie) {
    const match = c.match(/create-mod-auth=([^;]+)/);
    if (match) {
      token = match[1];
      break;
    }
  }

  // Fallback: try raw header parsing if getSetCookie is unavailable
  if (!token) {
    const raw = resp.headers.get('set-cookie') ?? '';
    const match = raw.match(/create-mod-auth=([^;]+)/);
    if (match) {
      token = match[1];
    }
  }

  if (!token) {
    throw new Error(
      `Failed to authenticate test user: status=${resp.status}, ` +
      `no create-mod-auth cookie in response`
    );
  }

  return token;
}

/**
 * Log in via the app's /login form POST so the browser context gets the
 * `create-mod-auth` cookie.
 *
 * Use this for tests that need a fully authenticated browser session.
 */
export async function login(page: Page, baseURL?: string): Promise<void> {
  const url = baseURL ?? appURL();

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
  const url = baseURL ?? appURL();
  const domain = new URL(url).hostname;

  const token = await authenticateUser();

  await page.context().addCookies([
    {
      name: 'create-mod-auth',
      value: token,
      domain,
      path: '/',
      httpOnly: true,
    },
  ]);
}
