import { test, expect } from '@playwright/test';

// Tests for the homepage tabbed sections (Latest, Trending, Highest Rated)
// and pagination within each tab.

test.describe('Homepage tabs', () => {
  test('shows Featured Builds section', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8090';
    await page.goto(url + '/');

    const featured = page.locator('text=Featured Builds');
    await expect(featured).toBeVisible();

    // Should have featured cards
    const featuredCards = page.locator('.card-featured');
    const count = await featuredCards.count();
    expect(count).toBeGreaterThanOrEqual(1);
    expect(count).toBeLessThanOrEqual(3);
  });

  test('shows all three tabs', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8090';
    await page.goto(url + '/');

    await expect(page.locator('a[role="tab"]:has-text("Latest")')).toBeVisible();
    await expect(page.locator('a[role="tab"]:has-text("Trending")')).toBeVisible();
    await expect(page.locator('a[role="tab"]:has-text("Highest Rated")')).toBeVisible();
  });

  test('Latest tab is active by default and shows schematics', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8090';
    await page.goto(url + '/');

    const latestTab = page.locator('a[role="tab"]:has-text("Latest")');
    await expect(latestTab).toHaveClass(/active/);

    const latestPane = page.locator('#tab-latest');
    await expect(latestPane).toHaveClass(/active/);

    // Should have schematic cards in the latest tab
    const cards = latestPane.locator('.card');
    const count = await cards.count();
    expect(count).toBeGreaterThan(0);
  });

  test('clicking Trending tab shows trending content', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8090';
    await page.goto(url + '/');

    // Click the Trending tab
    await page.locator('a[role="tab"]:has-text("Trending")').click();

    // Wait for tab pane to become active
    const trendingPane = page.locator('#tab-trending');
    await expect(trendingPane).toHaveClass(/active/, { timeout: 5000 });

    // Should have schematic cards
    const cards = trendingPane.locator('.card');
    const count = await cards.count();
    expect(count).toBeGreaterThan(0);
  });

  test('clicking Highest Rated tab shows highest rated content', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8090';
    await page.goto(url + '/');

    // Click the Highest Rated tab
    await page.locator('a[role="tab"]:has-text("Highest Rated")').click();

    // Wait for tab pane to become active
    const highestPane = page.locator('#tab-highest');
    await expect(highestPane).toHaveClass(/active/, { timeout: 5000 });

    // Should have schematic cards
    const cards = highestPane.locator('.card');
    const count = await cards.count();
    expect(count).toBeGreaterThan(0);
  });

  test('Latest tab has pagination and Next works via HTMX', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8090';
    await page.goto(url + '/');

    const latestPanel = page.locator('#tab-panel-latest');
    const nextBtn = latestPanel.locator('a:has-text("Next")');

    // Should have a Next button (more than 12 schematics in latest)
    await expect(nextBtn).toBeVisible();

    // Click Next — HTMX should swap the panel content
    await nextBtn.click();

    // Wait for the panel to update (page indicator should change to Page 2)
    await expect(latestPanel.locator('text=Page 2')).toBeVisible({ timeout: 5000 });

    // Previous button should now be enabled
    const prevBtn = latestPanel.locator('a:has-text("Previous"):not(.disabled)');
    await expect(prevBtn).toBeVisible();

    // Click Previous to go back to page 1
    await prevBtn.click();
    await expect(latestPanel.locator('text=Page 1')).toBeVisible({ timeout: 5000 });
  });

  test('HTMX tab pagination returns valid HTML', async ({ request, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8090';

    // Test each tab endpoint returns 200 with HX-Request header
    for (const tab of ['latest', 'trending', 'highest']) {
      const resp = await request.get(`${url}/?tab=${tab}&p=1`, {
        headers: { 'HX-Request': 'true' },
      });
      expect(resp.status(), `GET /?tab=${tab}&p=1 should return 200`).toBe(200);

      const html = await resp.text();
      expect(html).toContain(`tab-panel-${tab}`);
    }
  });
});
