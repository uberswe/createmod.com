const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });

  // Navigate directly with tag in URL
  await page.goto('http://localhost:8091/search?tag=train', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(2000);

  var pills = await page.evaluate(() =>
    Array.from(document.querySelectorAll('.search-filter-pill')).map(p => p.textContent.trim())
  );
  console.log('Pills with tag=train in URL:', pills);

  var activeFilters = await page.evaluate(() => {
    var el = document.querySelector('.search-active-filters');
    return el ? el.innerHTML.substring(0, 500) : 'NOT FOUND';
  });
  console.log('Active filters HTML:', activeFilters);

  await page.screenshot({ path: 'tests/screenshots/pills-direct.png' });

  await browser.close();
})();
