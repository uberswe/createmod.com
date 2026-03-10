import { test, expect } from '@playwright/test';

const BASE = process.env.APP_BASE_URL || 'https://createmod.com';

// Dismiss cookie/consent overlays that may block interaction
async function dismissOverlays(page: any) {
  // NitroPay consent modal "Accept" button
  const acceptBtn = page.locator('button:has-text("Accept")');
  if (await acceptBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
    await acceptBtn.click();
    await page.waitForTimeout(500);
  }
}

test.describe('Dropdown functionality after HTMX navigation', () => {

  test('language dropdown remains clickable after HTMX boosted navigation', async ({ page }) => {
    const consoleErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') consoleErrors.push(msg.text());
    });

    // Step 1: Full page load to schematic page
    await page.goto(`${BASE}/schematics/greenhousesqkk`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);
    await dismissOverlays(page);

    // Verify language dropdown works on initial load
    const langTrigger = page.locator('a.lang-flag[data-bs-toggle="dropdown"]').first();
    await expect(langTrigger).toBeVisible();

    const bootstrapAvailable = await page.evaluate(() => typeof (window as any).bootstrap !== 'undefined');
    expect(bootstrapAvailable).toBe(true);

    await langTrigger.click();
    const langMenu = langTrigger.locator('..').locator('.dropdown-menu');
    await expect(langMenu).toBeVisible({ timeout: 3000 });
    await page.keyboard.press('Escape');
    await page.waitForTimeout(300);

    // Step 2: Navigate to homepage via HTMX boosted link (brand link)
    await page.locator('a.top-header-brand').first().click();
    await page.waitForTimeout(3000);
    await page.waitForLoadState('networkidle');

    // Step 3: Navigate back to a schematic via HTMX boosted link
    const schematicLink = page.locator('a[href*="/schematics/"]').first();
    await expect(schematicLink).toBeVisible({ timeout: 10000 });
    await schematicLink.click();
    await page.waitForTimeout(3000);
    await page.waitForLoadState('networkidle');

    // Step 4: Check dropdown still works after HTMX navigation
    const langTrigger2 = page.locator('a.lang-flag[data-bs-toggle="dropdown"]').first();
    await expect(langTrigger2).toBeVisible();
    await langTrigger2.click();
    const langMenu2 = langTrigger2.locator('..').locator('.dropdown-menu');
    await expect(langMenu2).toBeVisible({ timeout: 5000 });

    if (consoleErrors.length > 0) {
      console.log('Console errors:', consoleErrors.slice(0, 5));
    }
  });

  test('dropdown works after index -> schematic -> index -> schematic cycle', async ({ page }) => {
    // Start at homepage
    await page.goto(`${BASE}/`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);
    await dismissOverlays(page);

    // Navigate to a schematic via HTMX
    const firstSchematic = page.locator('a[href*="/schematics/"]').first();
    await expect(firstSchematic).toBeVisible({ timeout: 10000 });
    await firstSchematic.click();
    await page.waitForTimeout(3000);

    // Navigate back to homepage via HTMX
    await page.locator('a.top-header-brand').first().click();
    await page.waitForTimeout(3000);

    // Navigate to schematic again via HTMX
    const secondSchematic = page.locator('a[href*="/schematics/"]').first();
    await expect(secondSchematic).toBeVisible({ timeout: 10000 });
    await secondSchematic.click();
    await page.waitForTimeout(3000);

    // Test the dropdown
    const langTrigger = page.locator('a.lang-flag[data-bs-toggle="dropdown"]').first();
    await expect(langTrigger).toBeVisible();
    await langTrigger.click();
    const langMenu = langTrigger.locator('..').locator('.dropdown-menu');
    await expect(langMenu).toBeVisible({ timeout: 5000 });
  });

  test('dropdown survives browser back/forward after HTMX navigation', async ({ page }) => {
    await page.goto(`${BASE}/schematics/greenhousesqkk`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);
    await dismissOverlays(page);

    // Navigate via HTMX
    await page.locator('a.top-header-brand').first().click();
    await page.waitForTimeout(3000);

    // Browser back
    await page.goBack();
    await page.waitForTimeout(3000);

    // Browser forward
    await page.goForward();
    await page.waitForTimeout(3000);

    // Browser back again
    await page.goBack();
    await page.waitForTimeout(3000);

    // Test the dropdown
    const langTrigger = page.locator('a.lang-flag[data-bs-toggle="dropdown"]').first();
    await expect(langTrigger).toBeVisible();
    await langTrigger.click();
    const langMenu = langTrigger.locator('..').locator('.dropdown-menu');
    await expect(langMenu).toBeVisible({ timeout: 5000 });
  });
});
