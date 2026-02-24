import { test, expect } from '@playwright/test';

// Verifies PocketBase service is healthy. Works in CI and local docker-compose.
// PB_URL should point to the PocketBase base URL.

test('PocketBase /api/health responds 200', async ({ request }) => {
  const pbUrl = process.env.PB_URL || 'http://localhost:8090';
  const resp = await request.get(`${pbUrl}/api/health`);
  expect(resp.status(), 'PocketBase health endpoint should return 200').toBe(200);
});
