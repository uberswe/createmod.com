import { test, expect } from '@playwright/test';

// Verifies the download interstitial page issues a one-time token
// and that the token is consumed after the first download attempt.
//
// Requires the "e2e-test-schematic" record seeded by global-setup.ts
// (moderated=true, deleted=null, non-paid).

test.describe('download interstitial token (single-use)', () => {
  test('interstitial issues single-use token and download succeeds once', async ({ page, request, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';

    // Navigate to the interstitial page for the seeded schematic
    const resp = await page.goto(url + '/get/e2e-test-schematic');
    expect(resp?.status(), 'interstitial page should load').toBe(200);

    // The page should contain a manual download link with a token
    const manualLink = page.locator('#manual-link');
    await expect(manualLink).toBeAttached();

    const href = await manualLink.getAttribute('href');
    expect(href, 'manual link href should exist').toBeTruthy();
    expect(href).toContain('/download/e2e-test-schematic?t=');

    // Extract the token from the href
    const tokenMatch = href!.match(/[?&]t=([a-f0-9]+)/);
    expect(tokenMatch, 'token should be a hex string').toBeTruthy();
    const token = tokenMatch![1];
    expect(token.length).toBeGreaterThan(0);

    // First request with the token should succeed (redirect to file or zip stream)
    const downloadResp = await request.get(url + `/download/e2e-test-schematic?t=${token}`);
    expect(
      downloadResp.status(),
      'first download with token should succeed',
    ).toBeLessThan(400);

    // Second request with the same token should fail (consumed)
    const replayResp = await request.get(url + `/download/e2e-test-schematic?t=${token}`);
    expect(replayResp.status(), 'replayed token should be rejected').toBe(403);
  });
});
