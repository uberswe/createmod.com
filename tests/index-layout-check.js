const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1400, height: 2000 } });
  await page.goto('http://localhost:8091/', { waitUntil: 'networkidle' });
  await page.waitForTimeout(2000);

  // Scroll to Latest section
  await page.evaluate(() => {
    var headings = document.querySelectorAll('h2');
    for (var i = 0; i < headings.length; i++) {
      if (headings[i].textContent.indexOf('Latest') !== -1) {
        headings[i].scrollIntoView();
        break;
      }
    }
  });
  await page.waitForTimeout(500);

  await page.screenshot({ path: 'tests/screenshots/index-latest-section.png' });

  var cards = await page.evaluate(function() {
    var cards = document.querySelectorAll('#tab-panel-latest .card, .tab-panel:nth-of-type(2) .card');
    if (cards.length === 0) cards = document.querySelectorAll('.card');
    return Array.from(cards).slice(0, 8).map(function(card, i) {
      var rect = card.getBoundingClientRect();
      var ratio = card.querySelector('.ratio');
      var ratioRect = ratio ? ratio.getBoundingClientRect() : null;
      var imgs = ratio ? ratio.querySelectorAll('img') : [];
      var imgInfo = Array.from(imgs).map(function(img) {
        var r = img.getBoundingClientRect();
        return {
          src: img.src.split('/').pop().substring(0, 30),
          naturalW: img.naturalWidth,
          naturalH: img.naturalHeight,
          displayW: Math.round(r.width),
          displayH: Math.round(r.height),
          position: window.getComputedStyle(img).position,
          overflow: window.getComputedStyle(img.parentElement).overflow
        };
      });
      return {
        idx: i,
        cardH: Math.round(rect.height),
        cardW: Math.round(rect.width),
        ratioH: ratioRect ? Math.round(ratioRect.height) : 'none',
        ratioOverflow: ratio ? window.getComputedStyle(ratio).overflow : 'none',
        images: imgInfo
      };
    });
  });
  console.log(JSON.stringify(cards, null, 2));

  await browser.close();
})();
