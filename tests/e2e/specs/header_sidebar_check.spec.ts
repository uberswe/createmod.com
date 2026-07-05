import { test, expect } from '@playwright/test';

// On desktop the top header is in normal flow and the left sidebar is fixed and
// full-height (starting at the very top). They share the top-left corner. When
// the rail is expanded (hover / pinned) it lifts ABOVE the header so the
// expanded menu covers the logo instead of sliding behind it.
test('expanded sidebar layers above the header (covers the logo)', async ({ page, baseURL }) => {
  const url = baseURL ?? 'http://localhost:8080';
  await page.goto(url + '/');

  // Take a screenshot of the top-left area showing header + sidebar
  await page.screenshot({ path: 'test-results/header-sidebar.png', clip: { x: 0, y: 0, width: 400, height: 300 } });

  const info = await page.evaluate(() => {
    const h = document.querySelector('.top-header') as HTMLElement | null;
    const s = document.querySelector('.sidebar-rail') as HTMLElement | null;
    if (!h || !s) return null;
    const hb = h.getBoundingClientRect();
    const sb = s.getBoundingClientRect();
    const headerZ = parseInt(window.getComputedStyle(h).zIndex) || 0;
    // Force the expanded state (matches :hover and the pinned rail) and read it.
    s.classList.add('sidebar-expanded');
    const expandedZ = parseInt(window.getComputedStyle(s).zIndex) || 0;
    return {
      headerTop: Math.round(hb.y),
      sidebarTop: Math.round(sb.y),
      headerZ,
      expandedZ,
    };
  });

  console.log('header/sidebar:', JSON.stringify(info));
  expect(info).not.toBeNull();
  if (info) {
    // Both anchor at the very top of the page.
    expect(info.headerTop).toBeLessThanOrEqual(1);
    expect(info.sidebarTop).toBeLessThanOrEqual(1);
    // Expanded: the rail lifts above the header so it covers the logo.
    expect(info.expandedZ).toBeGreaterThan(info.headerZ);
  }
});
