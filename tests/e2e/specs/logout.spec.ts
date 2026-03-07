import { test, expect } from '@playwright/test';
import { loginViaCookie } from '../helpers/auth';

// Covers both normal navigation and HTMX-triggered logout.
// Uses the backend cookie name: create-mod-auth (see internal/auth.CookieName)

test.describe('logout flows (normal + HTMX)', () => {
  test('GET /logout clears auth cookie and redirects (normal nav)', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';

    // Log in via cookie (fast) so we have an authenticated session
    await loginViaCookie(page, url);

    // Verify the cookie was set
    let cookies = await page.context().cookies();
    let authCookie = cookies.find(c => c.name === 'create-mod-auth');
    expect(authCookie, 'auth cookie should be set after login').toBeTruthy();
    expect(authCookie!.value).not.toBe('');

    // Navigate to /logout — the server clears the cookie and redirects to /
    await page.goto(url + '/logout');

    // After redirect, the auth cookie should be cleared
    cookies = await page.context().cookies();
    authCookie = cookies.find(c => c.name === 'create-mod-auth');
    // Cookie is either absent or has an empty value (MaxAge=-1 removes it)
    expect(authCookie?.value ?? '').toBe('');
  });

  test('HTMX logout returns HX-Redirect and 204', async ({ request, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';

    // Simulate HTMX request
    const resp = await request.get(url + '/logout', {
      headers: {
        'HX-Request': 'true',
      },
    });
    expect(resp.status()).toBe(204);

    // Validate HX-Redirect header
    const hxRedirect = resp.headers()['hx-redirect'];
    expect(hxRedirect).toBe('/');
  });
});
