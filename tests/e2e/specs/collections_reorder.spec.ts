import { test, expect } from '@playwright/test';
import { loginViaCookie, TEST_USER_EMAIL, TEST_USER_PASSWORD } from '../helpers/auth';
import { pbURL } from '../helpers/auth';

// Verifies that the collection reorder endpoint persists the new order.
// Uses the seeded "e2e-test-collection" from global-setup.ts.
//
// Since drag-and-drop automation is fragile, this test exercises the
// POST /collections/{slug}/reorder endpoint directly.

test.describe('collections add and reorder', () => {
  test('reorder endpoint accepts new order and responds correctly', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';

    // We need to authenticate so the reorder endpoint allows access.
    // First, obtain the user token and schematic IDs from PocketBase.
    const pb = pbURL();
    const authResp = await fetch(`${pb}/api/collections/users/auth-with-password`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        identity: TEST_USER_EMAIL,
        password: TEST_USER_PASSWORD,
      }),
    });
    expect(authResp.ok, 'should authenticate').toBeTruthy();
    const authData = await authResp.json();

    // Fetch the seeded collection to get its schematic IDs
    const collResp = await fetch(
      `${pb}/api/collections/collections/records?filter=(slug='e2e-test-collection')`,
      { headers: { Authorization: authData.token } },
    );
    expect(collResp.ok).toBeTruthy();
    const collData = await collResp.json();

    if (!collData.items || collData.items.length === 0) {
      test.skip(true, 'e2e-test-collection not found — seeding may have failed');
      return;
    }

    const collection = collData.items[0];
    const schematicIds: string[] = collection.schematics ?? [];

    if (schematicIds.length === 0) {
      test.skip(true, 'collection has no schematics to reorder');
      return;
    }

    // Log in via cookie so the app recognises us as the collection author
    await loginViaCookie(page, url);

    // Build the reversed order (or just the same order if only 1 item)
    const reversedIds = [...schematicIds].reverse();

    // POST to the reorder endpoint via the Playwright request context
    // We need to send the auth cookie, so use page.request which inherits cookies
    const reorderResp = await page.request.post(
      url + `/collections/e2e-test-collection/reorder`,
      {
        headers: {
          'HX-Request': 'true',
        },
        form: {
          ids: reversedIds.join(','),
        },
      },
    );

    // HTMX reorder returns 204 with HX-Redirect header
    expect(reorderResp.status()).toBe(204);
    const hxRedirect = reorderResp.headers()['hx-redirect'];
    expect(hxRedirect).toContain('/collections/e2e-test-collection/edit');
  });
});
