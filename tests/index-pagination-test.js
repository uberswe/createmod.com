const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
  await page.goto('http://localhost:8091/', { waitUntil: 'networkidle' });
  await page.waitForTimeout(2000);

  await page.screenshot({ path: 'tests/screenshots/pagination-before.png' });

  var before = await page.evaluate(function() {
    var panels = ['trending', 'latest', 'highest'];
    return panels.map(function(name) {
      var paginationDiv = document.getElementById('pagination-' + name);
      var pagination = paginationDiv ? paginationDiv.querySelector('.pagination') : null;
      var links = pagination ? pagination.querySelectorAll('a.page-link') : [];
      return {
        panel: name,
        hasPagination: !!pagination,
        inHeadingRow: paginationDiv ? paginationDiv.closest('.d-flex.align-items-center') !== null : false,
        linkTexts: Array.from(links).map(function(a) { return a.textContent.trim(); })
      };
    });
  });
  console.log('BEFORE:', JSON.stringify(before, null, 2));

  // Click "Next" on the first section that has a pagination link
  var clicked = await page.evaluate(function() {
    var panels = ['trending', 'latest', 'highest'];
    for (var i = 0; i < panels.length; i++) {
      var paginationDiv = document.getElementById('pagination-' + panels[i]);
      if (!paginationDiv) continue;
      var nextLink = paginationDiv.querySelector('a.page-link');
      if (nextLink) {
        nextLink.click();
        return panels[i];
      }
    }
    return null;
  });
  console.log('Clicked pagination on:', clicked);

  await page.waitForTimeout(3000);

  await page.screenshot({ path: 'tests/screenshots/pagination-after.png' });

  var after = await page.evaluate(function() {
    var panels = ['trending', 'latest', 'highest'];
    return panels.map(function(name) {
      var paginationDiv = document.getElementById('pagination-' + name);
      var pagination = paginationDiv ? paginationDiv.querySelector('.pagination') : null;
      var links = pagination ? pagination.querySelectorAll('a.page-link') : [];
      var pageItems = pagination ? pagination.querySelectorAll('.page-item.disabled .page-link') : [];
      var pageText = pageItems.length > 0 ? pageItems[0].textContent.trim() : 'none';
      return {
        panel: name,
        hasPagination: !!pagination,
        inHeadingRow: paginationDiv ? paginationDiv.closest('.d-flex.align-items-center') !== null : false,
        pageText: pageText,
        linkTexts: Array.from(links).map(function(a) { return a.textContent.trim(); })
      };
    });
  });
  console.log('AFTER:', JSON.stringify(after, null, 2));

  await browser.close();
})();
