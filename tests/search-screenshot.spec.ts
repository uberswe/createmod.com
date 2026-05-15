import { test } from '@playwright/test';

test('search page dark mode', async ({ page }) => {
  await page.goto('/search');
  await page.waitForLoadState('networkidle');
  await page.screenshot({ path: 'tests/screenshots/search-dark.png', fullPage: false });
});

test('search page dark mode with filters', async ({ page }) => {
  await page.goto('/search?category=flying-machines&mcv=1.20.X');
  await page.waitForLoadState('networkidle');
  await page.screenshot({ path: 'tests/screenshots/search-dark-filters.png', fullPage: false });
});

test('search page light mode', async ({ page }) => {
  await page.goto('/search');
  await page.waitForLoadState('networkidle');
  await page.evaluate(() => {
    (window as any).setTheme('light');
  });
  await page.waitForTimeout(500);
  await page.screenshot({ path: 'tests/screenshots/search-light.png', fullPage: false });
});

test('search page more filters expanded dark', async ({ page }) => {
  await page.goto('/search');
  await page.waitForLoadState('networkidle');
  const moreFilters = page.locator('.btn-more-filters');
  if (await moreFilters.count() > 0) {
    await moreFilters.click();
    await page.waitForTimeout(500);
  }
  await page.screenshot({ path: 'tests/screenshots/search-dark-more-filters.png', fullPage: false });
});

test('search page more filters expanded light', async ({ page }) => {
  await page.goto('/search');
  await page.waitForLoadState('networkidle');
  await page.evaluate(() => {
    (window as any).setTheme('light');
  });
  await page.waitForTimeout(300);
  const moreFilters = page.locator('.btn-more-filters');
  if (await moreFilters.count() > 0) {
    await moreFilters.click();
    await page.waitForTimeout(500);
  }
  await page.screenshot({ path: 'tests/screenshots/search-light-more-filters.png', fullPage: false });
});

test('search page light mode with filters', async ({ page }) => {
  await page.goto('/search?category=flying-machines&mcv=1.20.X');
  await page.waitForLoadState('networkidle');
  await page.evaluate(() => {
    (window as any).setTheme('light');
  });
  await page.waitForTimeout(500);
  await page.screenshot({ path: 'tests/screenshots/search-light-filters.png', fullPage: false });
});
