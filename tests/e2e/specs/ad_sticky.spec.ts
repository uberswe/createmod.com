import { test, expect } from '@playwright/test';

// The desktop right rail is built client-side by adrail.js: each page has a
// <div class="ad-rail ..." data-cm-adrail ...> container, and the helper appends
// a video ad plus the A/B variant's sticky ad unit:
//   Variant A -> a single ad whose id ends "_a_sticky"
//   Variant B -> a .ad-sticky-stack wrapper holding two display ads
// Either way the sticky element pins at top:8px while the video (and page)
// scroll. Uses /explore because it always renders an .ad-rail.
// In CI no real ad scripts load, but the helper still builds the DOM (it only
// needs the nitroAds stub), so we can verify the computed sticky styles.

test.describe('Ad rail stickiness', () => {
  test.use({ viewport: { width: 1400, height: 900 } });

  test('the ad rail builds a sticky ad unit pinned near the top', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/explore', { waitUntil: 'domcontentloaded' });

    const adRail = page.locator('.ad-rail[data-cm-adrail]').first();
    await expect(adRail).toBeAttached({ timeout: 10000 });

    // adrail.js builds the rail on load — wait for the sticky unit to appear.
    await page.waitForFunction(() => {
      const r = document.querySelector('.ad-rail[data-cm-adrail]');
      return !!(r && r.querySelector(':scope > [id$="_a_sticky"], :scope > .ad-sticky-stack'));
    }, { timeout: 10000 });

    const result = await adRail.evaluate((el) => {
      const sticky = el.querySelector(':scope > [id$="_a_sticky"], :scope > .ad-sticky-stack');
      const ss = sticky ? window.getComputedStyle(sticky) : null;
      const video = el.querySelector('[id$="_video"]');
      const vs = video ? window.getComputedStyle(video) : null;
      return {
        hasSticky: !!sticky,
        stickyPosition: ss?.position,
        stickyTop: ss?.top,
        videoNotSticky: vs ? vs.position !== 'sticky' : true,
      };
    });

    expect(result.hasSticky).toBe(true);
    expect(result.stickyPosition).toBe('sticky');
    expect(result.stickyTop).toBe('8px');
    // The video slot scrolls with the page (not sticky).
    expect(result.videoNotSticky).toBe(true);
  });

  test('ad rail outer container has correct layout styles', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/explore', { waitUntil: 'domcontentloaded' });

    const adRail = page.locator('.ad-rail').first();
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
    // The outer container is not sticky itself — the inner unit handles stickiness
    expect(outerStyles.position).not.toBe('sticky');
  });
});
