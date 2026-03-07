import { test, expect } from '@playwright/test';

test.describe('hx-boost does not break non-HTML links', () => {

  test('clicking schematic image opens lightbox, not encoded data', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    const resp = await page.goto(url + '/schematics/industrial-estate');

    if (resp?.status() === 404) {
      test.skip(true, 'schematic "industrial-estate" not found — not enough seed data');
      return;
    }

    // Wait for the main image link
    const imageLink = page.locator('a[data-fslightbox="gallery"]').first();
    if (!(await imageLink.isVisible().catch(() => false))) {
      test.skip(true, 'no gallery image found on schematic page');
      return;
    }

    // Verify hx-boost="false" is set
    const boost = await imageLink.getAttribute('hx-boost');
    expect(boost).toBe('false');

    // Click the image
    await imageLink.click();

    // Wait a moment for lightbox or navigation
    await page.waitForTimeout(1500);

    // The page should NOT have navigated to show raw image data.
    // If hx-boost intercepted it, the body would contain raw image bytes
    // or an error page. The URL should still be the schematic page.
    const currentUrl = page.url();
    expect(currentUrl, 'should stay on schematic page (lightbox opens in overlay)').toContain('/schematics/');

    // The page body should NOT contain raw data indicators
    const bodyText = await page.locator('body').innerText();
    expect(bodyText).not.toContain('PNG');
    expect(bodyText).not.toContain('JFIF');
  });

  test('clicking download button starts download, not encoded data', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    const resp = await page.goto(url + '/get/house-with-swappable-rooms');

    if (resp?.status() === 404) {
      test.skip(true, 'schematic "house-with-swappable-rooms" not found — not enough seed data');
      return;
    }

    const manualLink = page.locator('#manual-link, #manual-link-ext').first();
    if (!(await manualLink.isVisible().catch(() => false))) {
      test.skip(true, 'no manual download link found');
      return;
    }

    // Verify hx-boost="false" is set
    const boost = await manualLink.getAttribute('hx-boost');
    expect(boost).toBe('false');

    // Set up download listener
    const downloadPromise = page.waitForEvent('download', { timeout: 5000 }).catch(() => null);

    // Click the download button
    await manualLink.click();
    await page.waitForTimeout(2000);

    // The page body should NOT contain raw file data
    const bodyText = await page.locator('body').innerText();
    expect(bodyText).not.toContain('PK'); // zip magic bytes
    // Should not show encoded binary data in the page
    expect(bodyText.length, 'body should not be filled with binary data').toBeLessThan(10000);
  });

  test('download timer stops after manual click', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';

    // Track requests to /download/
    const downloadRequests: string[] = [];
    page.on('request', (req) => {
      if (req.url().includes('/download/')) {
        downloadRequests.push(req.url());
      }
    });

    const resp = await page.goto(url + '/get/house-with-swappable-rooms');
    if (resp?.status() === 404) {
      test.skip(true, 'schematic "house-with-swappable-rooms" not found — not enough seed data');
      return;
    }

    const manualLink = page.locator('#manual-link, #manual-link-ext').first();
    if (!(await manualLink.isVisible().catch(() => false))) {
      test.skip(true, 'no manual download link found');
      return;
    }

    // Click manual download after 2 seconds (timer still has ~8 seconds left)
    await page.waitForTimeout(2000);
    await manualLink.click();

    // Wait longer than the remaining countdown (8+ seconds)
    await page.waitForTimeout(10000);

    // Should have exactly 1 download request (from the click, not the timer)
    console.log('Download requests:', downloadRequests.length);
    expect(downloadRequests.length, 'timer should not fire after manual click').toBe(1);
  });

  test('download timer stops when navigating away via HTMX', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';

    // Track requests to /download/
    const downloadRequests: string[] = [];
    page.on('request', (req) => {
      if (req.url().includes('/download/')) {
        downloadRequests.push(req.url());
      }
    });

    const resp = await page.goto(url + '/get/house-with-swappable-rooms');
    if (resp?.status() === 404) {
      test.skip(true, 'schematic "house-with-swappable-rooms" not found — not enough seed data');
      return;
    }

    const countdown = page.locator('#countdown, #countdown-ext').first();
    if (!(await countdown.isVisible().catch(() => false))) {
      test.skip(true, 'no countdown element found');
      return;
    }

    // Wait 2 seconds then navigate away using a sidebar/header link (hx-boosted)
    await page.waitForTimeout(2000);

    // Click the site logo/home link to navigate away via HTMX
    const homeLink = page.locator('a[href="/"]').first();
    if (await homeLink.isVisible()) {
      await homeLink.click();
      await page.waitForTimeout(12000);

      // Timer should have been cancelled — no download request should have fired
      console.log('Download requests after navigating away:', downloadRequests.length);
      expect(downloadRequests.length, 'timer should not fire after navigating away').toBe(0);
    }
  });
});
