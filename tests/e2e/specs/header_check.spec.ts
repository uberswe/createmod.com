import { test, expect } from '@playwright/test';

test('header logo is vertically centered', async ({ page, baseURL }) => {
  const url = baseURL ?? 'http://localhost:8090';
  await page.goto(url + '/');

  // Screenshot the header
  const header = page.locator('.top-header');
  await header.screenshot({ path: 'test-results/header.png' });

  // Get header and logo dimensions
  const headerBox = await header.boundingBox();
  const logo = page.locator('.top-header-brand img');
  const logoBox = await logo.boundingBox();

  console.log('Header:', JSON.stringify(headerBox));
  console.log('Logo:', JSON.stringify(logoBox));

  if (headerBox && logoBox) {
    const topSpace = logoBox.y - headerBox.y;
    const bottomSpace = (headerBox.y + headerBox.height) - (logoBox.y + logoBox.height);
    console.log(`Top space: ${topSpace}px, Bottom space: ${bottomSpace}px, Difference: ${Math.abs(topSpace - bottomSpace)}px`);

    // Logo should be roughly centered (within 4px tolerance)
    expect(Math.abs(topSpace - bottomSpace)).toBeLessThan(4);
  }
});
