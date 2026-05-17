const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
  page.on('pageerror', err => console.log('PAGE ERROR:', err.message));

  console.log('=== Test 1: Tag selection triggers filter ===');
  await page.goto('http://localhost:8091/search', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(3000);

  // Open More Filters
  await page.click('.btn-more-filters');
  await page.waitForTimeout(1000);

  // Select a tag via Tom Select
  await page.evaluate(() => document.getElementById('search-more-filters').scrollIntoView());
  var tagControl = await page.$('.ts-wrapper .ts-control');
  await tagControl.click();
  await page.waitForTimeout(300);
  await page.keyboard.type('train');
  await page.waitForTimeout(800);
  await page.screenshot({ path: 'tests/screenshots/t01-tag-dropdown.png' });

  // Monitor HTMX request
  var htmxFired = false;
  await page.evaluate(() => {
    window._htmxRequests = [];
    document.addEventListener('htmx:beforeRequest', function(e) {
      window._htmxRequests.push(e.detail.requestConfig?.path || 'unknown');
    });
  });

  // Click first dropdown option
  await page.click('.ts-dropdown .option');
  await page.waitForTimeout(500);

  // Check if HTMX fired
  var requests = await page.evaluate(() => window._htmxRequests);
  console.log('HTMX requests after tag select:', requests);

  // Check the current tag select value
  var tagValue = await page.evaluate(() => {
    var el = document.getElementById('search-tag-select');
    if (!el) return 'not found';
    if (el.tomselect) return 'TS items: ' + JSON.stringify(el.tomselect.items);
    return 'values: ' + Array.from(el.selectedOptions).map(o => o.value).join(',');
  });
  console.log('Tag select value:', tagValue);

  // Wait for HTMX response
  await page.waitForTimeout(3000);
  await page.screenshot({ path: 'tests/screenshots/t02-after-tag-select.png' });

  // Check URL
  var url = page.url();
  console.log('URL after tag select:', url);

  // Check pills
  var pills = await page.evaluate(() =>
    Array.from(document.querySelectorAll('.search-filter-pill')).map(p => p.textContent.trim())
  );
  console.log('Active pills:', pills);

  console.log('\n=== Test 2: View mode in pagination URLs ===');
  await page.goto('http://localhost:8091/search?view=list', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(2000);

  // Check if list view is active
  var viewMode = await page.evaluate(() => document.getElementById('search-view-input')?.value);
  console.log('View mode:', viewMode);

  // Check pagination URLs
  var paginationUrls = await page.evaluate(() => {
    var links = document.querySelectorAll('.pagination a.page-link');
    return Array.from(links).map(a => a.getAttribute('hx-get') || a.getAttribute('href'));
  });
  console.log('Pagination URLs:', paginationUrls);
  var allHaveView = paginationUrls.every(u => u && u.includes('view=list'));
  console.log('All pagination URLs include view=list:', allHaveView);

  // Check list cards rendered
  var listCards = await page.evaluate(() => document.querySelectorAll('.search-list-item').length);
  console.log('List view cards:', listCards);
  await page.screenshot({ path: 'tests/screenshots/t03-list-view.png' });

  console.log('\n=== Test 3: Bottom pagination ===');
  await page.goto('http://localhost:8091/search', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(2000);
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
  await page.waitForTimeout(500);
  await page.screenshot({ path: 'tests/screenshots/t04-bottom.png' });
  var bottomPag = await page.evaluate(() => {
    var navs = document.querySelectorAll('nav[aria-label="Search results pagination"]');
    return navs.length;
  });
  console.log('Pagination nav elements:', bottomPag);

  console.log('\n=== Test 4: Infinite scroll mode ===');
  await page.goto('http://localhost:8091/search?per_page=infinite', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(2000);
  var infiniteInfo = await page.evaluate(() => {
    return {
      loadMoreBtn: !!document.getElementById('load-more-btn'),
      loadMoreTag: document.getElementById('load-more-btn')?.tagName,
      bottomPagination: document.querySelectorAll('nav[aria-label="Search results pagination"]').length,
      inlinePagination: document.querySelectorAll('.search-results-controls .pagination').length,
      perPageValue: document.querySelector('.search-per-page-select')?.value,
    };
  });
  console.log('Infinite scroll mode:', JSON.stringify(infiniteInfo));
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
  await page.waitForTimeout(500);
  await page.screenshot({ path: 'tests/screenshots/t05-infinite.png' });

  console.log('\n=== Test 5: Per-page select styling ===');
  await page.goto('http://localhost:8091/search', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(2000);
  var perPageStyles = await page.evaluate(() => {
    var el = document.querySelector('.search-per-page-select');
    if (!el) return null;
    var cs = getComputedStyle(el);
    var viewBtns = document.querySelectorAll('.view-btn');
    var btnH = viewBtns.length ? viewBtns[0].offsetHeight : 0;
    return {
      height: el.offsetHeight,
      width: el.offsetWidth,
      fontSize: cs.fontSize,
      viewBtnHeight: btnH,
      aligned: Math.abs(el.offsetHeight - btnH) <= 4,
    };
  });
  console.log('Per-page select:', JSON.stringify(perPageStyles));

  await browser.close();
  console.log('\n=== All tests complete ===');
})();
