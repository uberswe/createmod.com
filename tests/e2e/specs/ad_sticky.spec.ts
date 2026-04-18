import { test, expect } from '@playwright/test';

// Test that the ad rail CSS is correctly applied so NitroPay sticky-stack works.
// The outer .ad-rail uses align-self:stretch for full height,
// while the inner > div uses position:sticky to stay visible during scroll.
// Uses /explore because it always renders .ad-rail (the homepage does not have one).
// In CI no ad scripts load, so we verify computed styles rather than visible content.

test.describe('Ad rail stickiness', () => {
  test.use({ viewport: { width: 1400, height: 900 } });

  test('ad rail inner divs have sticky positioning', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/explore', { waitUntil: 'domcontentloaded' });

    // The .ad-rail is hidden below xl breakpoint via d-none/d-xl-block.
    // At 1400px wide it should be present in the DOM.
    const adRail = page.locator('.ad-rail').first();
    await expect(adRail).toBeAttached({ timeout: 10000 });

    // Verify the inner child divs have position: sticky applied via CSS
    const innerStyles = await adRail.evaluate((el) => {
      const children = el.querySelectorAll(':scope > div');
      return Array.from(children).map((child) => {
        const style = window.getComputedStyle(child);
        return { position: style.position, top: style.top };
      });
    });

    expect(innerStyles.length).toBeGreaterThan(0);
    for (const style of innerStyles) {
      expect(style.position).toBe('sticky');
      expect(style.top).toBe('110px');
    }
  });

  test('ad rail outer container has correct layout styles', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/explore', { waitUntil: 'domcontentloaded' });

    const adRail = page.locator('.ad-rail').first();
    await expect(adRail).toBeAttached({ timeout: 10000 });

    // Verify the outer .ad-rail has the correct CSS for stretch layout
    const outerStyles = await adRail.evaluate((el) => {
      const style = window.getComputedStyle(el);
      return {
        alignSelf: style.alignSelf,
        width: style.width,
        flexShrink: style.flexShrink,
        position: style.position,
      };
    });

    // align-self: stretch ensures the rail fills its flex parent height
    expect(outerStyles.alignSelf).toBe('stretch');
    // width should be 300px
    expect(outerStyles.width).toBe('300px');
    // flex-shrink: 0 prevents the rail from collapsing
    expect(outerStyles.flexShrink).toBe('0');
    // The outer container should NOT be sticky itself — the inner child handles stickiness
    expect(outerStyles.position).not.toBe('sticky');
  });
});
