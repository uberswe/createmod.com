const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });

  await page.goto('http://localhost:8091/generators/hull', { waitUntil: 'networkidle' });
  await page.waitForTimeout(2000);

  const presets = [
    'sloop', 'schooner', 'frigate', 'galleon', 'carrack', 'longship',
    'clipper', 'shipOfTheLine', 'cog', 'pinnace', 'dhow', 'junk',
    'warGalley', 'flyingShip', 'dragonShip', 'ark',
    'trawler', 'tugboat', 'destroyer', 'speedboat', 'yacht'
  ];

  for (const preset of presets) {
    await page.click(`button.gen-preset[data-preset="${preset}"]`);
    await page.waitForTimeout(1500);
    await page.screenshot({
      path: `tests/screenshots/hull-${preset}.png`,
      clip: { x: 0, y: 0, width: 1400, height: 900 }
    });
    console.log(`Captured: ${preset}`);
  }

  await browser.close();
  console.log('Done — all hull preset screenshots saved.');
})();
