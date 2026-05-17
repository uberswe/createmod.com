import { test, expect } from '@playwright/test';

const GUIDE_URL = 'http://localhost:8091/generators/balloon/YjEuMzYuMTguMTYuMC4wLjAuMC4wLjEuMS4wLjQuMC4xLjAuNC41Lncud2gudy5z/guide';

test('guide page full screenshot', async ({ page }) => {
  await page.goto(GUIDE_URL);
  await page.waitForSelector('#guide-canvas', { timeout: 15000 });
  // Wait for render to complete
  await page.waitForTimeout(2000);
  await page.screenshot({ path: 'tests/screenshots/guide-full.png', fullPage: true });
});

test('guide page - navigate layers and screenshot each', async ({ page }) => {
  await page.goto(GUIDE_URL);
  await page.waitForSelector('#guide-canvas', { timeout: 15000 });
  await page.waitForTimeout(2000);

  // Screenshot layer 1
  await page.screenshot({ path: 'tests/screenshots/guide-layer1.png', fullPage: true });

  // Go to layer 2
  const nextBtn = page.locator('#guide-next');
  if (await nextBtn.isEnabled()) {
    await nextBtn.click();
    await page.waitForTimeout(500);
    await page.screenshot({ path: 'tests/screenshots/guide-layer2.png', fullPage: true });
  }

  // Go to layer 5
  for (let i = 0; i < 3; i++) {
    if (await nextBtn.isEnabled()) {
      await nextBtn.click();
      await page.waitForTimeout(200);
    }
  }
  await page.waitForTimeout(500);
  await page.screenshot({ path: 'tests/screenshots/guide-layer5.png', fullPage: true });

  // Go to last layer
  const slider = page.locator('#guide-slider');
  const max = await slider.getAttribute('max');
  if (max) {
    await slider.fill(max);
    await slider.dispatchEvent('input');
    await page.waitForTimeout(500);
    await page.screenshot({ path: 'tests/screenshots/guide-last-layer.png', fullPage: true });
  }
});

test('guide page viewport sizes', async ({ page }) => {
  // Desktop wide
  await page.setViewportSize({ width: 1400, height: 900 });
  await page.goto(GUIDE_URL);
  await page.waitForSelector('#guide-canvas', { timeout: 15000 });
  await page.waitForTimeout(2000);
  await page.screenshot({ path: 'tests/screenshots/guide-desktop-wide.png', fullPage: true });

  // Desktop narrow
  await page.setViewportSize({ width: 1024, height: 768 });
  await page.goto(GUIDE_URL);
  await page.waitForSelector('#guide-canvas', { timeout: 15000 });
  await page.waitForTimeout(2000);
  await page.screenshot({ path: 'tests/screenshots/guide-desktop-narrow.png', fullPage: true });

  // Mobile
  await page.setViewportSize({ width: 375, height: 812 });
  await page.goto(GUIDE_URL);
  await page.waitForSelector('#guide-canvas', { timeout: 15000 });
  await page.waitForTimeout(2000);
  await page.screenshot({ path: 'tests/screenshots/guide-mobile.png', fullPage: true });
});
