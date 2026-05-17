const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
  await page.goto('http://localhost:8091/', { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);
  // Screenshot a single card at high res
  const card = await page.$('.card');
  if (card) {
    await card.screenshot({ path: 'tests/screenshots/index-single-card.png' });
  }
  // Also check if any cards have images that are taller than the ratio box
  const result = await page.evaluate(() => {
    const cards = document.querySelectorAll('.card-body-borderless.ratio');
    const info = [];
    cards.forEach((c, i) => {
      const rect = c.getBoundingClientRect();
      const imgs = c.querySelectorAll('img');
      const imgInfo = [];
      imgs.forEach(img => {
        imgInfo.push({
          src: img.src.split('/').pop(),
          naturalW: img.naturalWidth,
          naturalH: img.naturalHeight,
          displayW: img.getBoundingClientRect().width,
          displayH: img.getBoundingClientRect().height,
          visible: img.style.display !== 'none'
        });
      });
      info.push({ idx: i, containerW: rect.width, containerH: rect.height, images: imgInfo });
    });
    return info;
  });
  console.log(JSON.stringify(result, null, 2));
  await browser.close();
})();
