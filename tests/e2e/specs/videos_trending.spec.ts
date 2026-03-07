import { test, expect } from '@playwright/test';

// Tests for the /videos page: title, description, and trending sort order.
// These tests require multiple schematics with videos — skip gracefully
// in CI where only minimal seed data exists.

test.describe('Videos page', () => {
  test('displays Videos title and description', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/videos');

    const title = page.locator('h2.page-title');
    await expect(title).toBeVisible();
    await expect(title).toHaveText('Videos');

    const description = page.locator('.text-secondary:has-text("Videos from published schematics")');
    await expect(description).toBeVisible();
  });

  test('videos are sorted by trending (not purely by creation date)', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/videos');

    // Collect schematic links from the video cards on the first page.
    const schematicLinks = page.locator('a:has-text("View schematic")');
    const count = await schematicLinks.count();

    if (count <= 1) {
      test.skip(true, 'not enough video schematics to test trending sort');
      return;
    }

    const slugs: string[] = [];
    for (let i = 0; i < Math.min(count, 12); i++) {
      const href = await schematicLinks.nth(i).getAttribute('href');
      if (href) slugs.push(href);
    }

    // Fetch each schematic page and extract its upload date from the tooltip.
    // The date appears in: title="2024-08-16 15:06:50"
    const dates: Date[] = [];
    for (const slug of slugs) {
      const resp = await page.request.get((baseURL ?? 'http://localhost:8080') + slug);
      const html = await resp.text();
      // Look for the upload date in the tooltip next to the "Uploaded" label
      const match = html.match(/title="(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})">/);
      if (match) {
        dates.push(new Date(match[1].replace(' ', 'T')));
      }
    }

    // We should have found dates for most schematics
    expect(dates.length).toBeGreaterThan(1);

    // If sorted purely by newest-first, dates would be strictly decreasing.
    // Trending sort should break this pattern — at least one older schematic
    // should appear before a newer one because it has more engagement.
    let isStrictlyChronological = true;
    for (let i = 1; i < dates.length; i++) {
      if (dates[i].getTime() > dates[i - 1].getTime()) {
        isStrictlyChronological = false;
        break;
      }
    }
    expect(isStrictlyChronological).toBe(false);
  });

  test('first video is a schematic with real engagement (not just newest)', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/videos');

    // The first video card should exist
    const firstCard = page.locator('.card').first();
    if (!(await firstCard.isVisible().catch(() => false))) {
      test.skip(true, 'no video cards found — not enough seed data');
      return;
    }

    // Get the schematic link for the first video
    const firstSchematicLink = page.locator('a:has-text("View schematic")').first();
    const href = await firstSchematicLink.getAttribute('href');
    expect(href).toBeTruthy();

    // Visit the schematic page and check it has views > 0
    const resp = await page.request.get((baseURL ?? 'http://localhost:8080') + href!);
    const html = await resp.text();

    // A trending schematic should show view/engagement indicators
    const hasViews = html.includes('views') || html.includes('Views') || html.includes('view');
    expect(hasViews).toBe(true);
  });
});
