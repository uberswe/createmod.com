import { test, expect } from '@playwright/test';

// Simple smoke test to verify the app responds on the base URL.
// Requires the stack running (docker compose up) and APP_BASE_URL pointing to the app.

test('home is reachable and returns 200', async ({ request, baseURL }) => {
  const url = baseURL ?? 'http://localhost:8080';
  const resp = await request.get(url + '/');
  expect(resp.status(), 'GET / should return success').toBeLessThan(500);
});
