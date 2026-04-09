import { test, expect } from '@playwright/test';

// Test that the ad rail inner div is sticky when scrolling on desktop.
// The outer .ad-rail uses align-self:stretch for full height (NitroPay needs it),
// while the inner > div uses position:sticky to stay visible during scroll.

test.describe('Ad rail stickiness', () => {
  test.use({ viewport: { width: 1400, height: 900 } });

  test('ad rail inner div stays visible after scrolling on homepage', async ({ page }) => {
    await page.goto('https://createmod.com/', { waitUntil: 'domcontentloaded' });

    const adRail = page.locator('.ad-rail').first();
    await expect(adRail).toBeVisible({ timeout: 10000 });

    // The inner div is the direct child of .ad-rail
    const innerDiv = adRail.locator('> div').first();
    await expect(innerDiv).toBeVisible();

    // Get initial position of the inner div
    const initialBox = await innerDiv.boundingBox();
    expect(initialBox).toBeTruthy();

    // Scroll down significantly
    await page.evaluate(() => window.scrollBy(0, 1500));
    await page.waitForTimeout(500);

    // Check the inner div's computed position style
    const position = await innerDiv.evaluate((el) => {
      return window.getComputedStyle(el).position;
    });
    console.log(`Inner div computed position: ${position}`);
    expect(position).toBe('sticky');

    // Get position after scroll - sticky element should remain in viewport
    const afterScrollBox = await innerDiv.boundingBox();
    expect(afterScrollBox).toBeTruthy();

    console.log(`Before scroll - top: ${initialBox!.y}, After scroll - top: ${afterScrollBox!.y}`);

    // A sticky element should stay in the viewport (positive y, within viewport height)
    expect(afterScrollBox!.y).toBeGreaterThanOrEqual(0);
    expect(afterScrollBox!.y).toBeLessThan(900);
  });

  test('ad rail outer container fills parent height', async ({ page }) => {
    await page.goto('https://createmod.com/', { waitUntil: 'domcontentloaded' });

    const adRail = page.locator('.ad-rail').first();
    await expect(adRail).toBeVisible({ timeout: 10000 });

    // Check outer ad-rail uses align-self: stretch
    const alignSelf = await adRail.evaluate((el) => {
      return window.getComputedStyle(el).alignSelf;
    });
    console.log(`Ad rail align-self: ${alignSelf}`);

    // Get the ad rail height vs its parent flex container height
    const heights = await adRail.evaluate((el) => {
      const parent = el.closest('.d-flex') as HTMLElement;
      return {
        adRailHeight: el.getBoundingClientRect().height,
        parentHeight: parent ? parent.getBoundingClientRect().height : 0,
      };
    });
    console.log(`Ad rail height: ${heights.adRailHeight}, Parent height: ${heights.parentHeight}`);

    // The ad rail should be close to the parent height (stretch behavior)
    if (heights.parentHeight > 0) {
      const ratio = heights.adRailHeight / heights.parentHeight;
      console.log(`Height ratio (ad/parent): ${ratio}`);
      expect(ratio).toBeGreaterThan(0.8);
    }

    // The outer container should NOT be sticky itself
    const outerPosition = await adRail.evaluate((el) => {
      return window.getComputedStyle(el).position;
    });
    console.log(`Outer ad-rail position: ${outerPosition}`);
    // Should be static (not sticky) — the inner child handles stickiness
    expect(outerPosition).not.toBe('sticky');
  });
});
