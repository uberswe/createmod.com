import { test, expect } from '@playwright/test';

// The desktop right rail is built client-side by adrail.js: each page has a
// <div class="cm-side-rail ..." data-cm-adrail ...> container, and the helper
// appends a single NitroPay sticky-stack unit whose id ends "_sticky". The
// holder itself is NOT sticky — NitroPay pins individual ads inside it — but
// it must span the full rail column (height: 100%) so ads can be placed down
// the whole page height. Uses /explore because it always renders an .cm-side-rail.
// In CI no real ad scripts load, but the helper still builds the DOM (it only
// needs the nitroAds stub), so we can verify the computed styles.

test.describe('Ad rail', () => {
  test.use({ viewport: { width: 1400, height: 900 } });

  test('the ad rail builds a full-height sticky-stack holder', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/explore', { waitUntil: 'domcontentloaded' });

    const adRail = page.locator('.cm-side-rail[data-cm-adrail]').first();
    await expect(adRail).toBeAttached({ timeout: 10000 });

    // adrail.js builds the rail on load — wait for the unit holder to appear.
    await page.waitForFunction(() => {
      const r = document.querySelector('.cm-side-rail[data-cm-adrail]');
      return !!(r && r.querySelector(':scope > [id$="_sticky"]'));
    }, { timeout: 10000 });

    const result = await adRail.evaluate((el) => {
      const holder = el.querySelector(':scope > [id$="_sticky"]');
      const hs = holder ? window.getComputedStyle(holder) : null;
      return {
        hasHolder: !!holder,
        // NitroPay manages stickiness internally; the holder must not be
        // sticky itself or the stack can't place ads down the column.
        holderPosition: hs?.position,
        holderHeight: holder ? (holder as HTMLElement).offsetHeight : 0,
        railHeight: (el as HTMLElement).offsetHeight,
        // The old video slot must be gone — video runs through the global
        // floating outstream player now.
        hasVideoSlot: !!el.querySelector('[id$="_video"]'),
      };
    });

    expect(result.hasHolder).toBe(true);
    expect(result.holderPosition).not.toBe('sticky');
    // height: 100% — the holder spans the full rail column.
    expect(result.holderHeight).toBeGreaterThan(0);
    expect(Math.abs(result.holderHeight - result.railHeight)).toBeLessThanOrEqual(20);
    expect(result.hasVideoSlot).toBe(false);
  });

  test('ad rail outer container has correct layout styles', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/explore', { waitUntil: 'domcontentloaded' });

    const adRail = page.locator('.cm-side-rail').first();
    await expect(adRail).toBeAttached({ timeout: 10000 });

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
    // The outer container is not sticky itself — NitroPay handles pinning
    expect(outerStyles.position).not.toBe('sticky');
  });
});
