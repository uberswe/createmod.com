const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });

  // Styleguide forms section
  await page.goto('http://localhost:8091/styleguide#forms', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(2000);
  await page.evaluate(() => document.getElementById('forms').scrollIntoView());
  await page.waitForTimeout(500);
  await page.screenshot({ path: 'tests/screenshots/styleguide-forms.png' });

  // Scroll down to see range sliders and search filter bar
  await page.evaluate(() => window.scrollBy(0, 800));
  await page.waitForTimeout(500);
  await page.screenshot({ path: 'tests/screenshots/styleguide-sliders.png' });

  // Search page with filters expanded
  await page.goto('http://localhost:8091/search', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(2000);
  // Click "More filters" to expand
  var moreBtn = await page.$('.btn-more-filters');
  if (moreBtn) await moreBtn.click();
  await page.waitForTimeout(500);
  await page.screenshot({ path: 'tests/screenshots/search-filters.png' });

  await browser.close();
})();
