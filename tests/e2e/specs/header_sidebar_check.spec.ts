import { test, expect } from '@playwright/test';

test('header and sidebar do not overlap', async ({ page, baseURL }) => {
  const url = baseURL ?? 'http://localhost:8090';
  await page.goto(url + '/');

  // Take a screenshot of the top-left area showing header + sidebar
  await page.screenshot({ path: 'test-results/header-sidebar.png', clip: { x: 0, y: 0, width: 400, height: 300 } });

  // Verify the sidebar starts below the header
  const header = page.locator('.top-header');
  const sidebar = page.locator('.sidebar-rail');
  const headerBox = await header.boundingBox();
  const sidebarBox = await sidebar.boundingBox();

  console.log('Header bottom:', headerBox ? headerBox.y + headerBox.height : 'N/A');
  console.log('Sidebar top:', sidebarBox?.y);

  if (headerBox && sidebarBox) {
    // Sidebar should start at or below the header bottom
    expect(sidebarBox.y).toBeGreaterThanOrEqual(headerBox.y + headerBox.height);
  }
});
