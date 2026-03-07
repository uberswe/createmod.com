import { test, expect } from '@playwright/test';

// Verifies HTMX logout behavior without requiring authentication or seeds.
// Mirrors internal/pages/logout_http_test.go expectations:
// - Status 204 No Content
// - HX-Redirect: /

test('HTMX logout returns HX-Redirect and 204', async ({ request, baseURL }) => {
  const url = baseURL ?? 'http://localhost:8080';

  const resp = await request.get(url + '/logout', {
    headers: {
      'HX-Request': 'true',
    },
  });

  expect(resp.status()).toBe(204);
  const hxRedirect = resp.headers()['hx-redirect'];
  expect(hxRedirect).toBe('/');
});
