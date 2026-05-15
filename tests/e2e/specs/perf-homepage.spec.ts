import { test, expect } from '@playwright/test';

test('Homepage load performance via schematic navigation', async ({ page }) => {
  const BASE = 'https://createmod.com';

  // 1. Go to the index page (full navigation)
  console.log('=== Step 1: Navigate to homepage (cold) ===');
  const startIndex = Date.now();
  await page.goto(BASE + '/', { waitUntil: 'load' });
  const indexLoadTime = Date.now() - startIndex;
  console.log(`Homepage initial load: ${indexLoadTime}ms`);

  // Wait for lazy-loaded images
  await page.waitForTimeout(3000);

  const initialPerf = await getPagePerf(page);
  printPerf('Initial Homepage', initialPerf);

  // 2. Navigate to schematic page (full navigation to avoid HTMX issues)
  console.log('\n=== Step 2: Navigate to schematic page ===');
  const startSchematic = Date.now();
  await page.goto(BASE + '/schematics/aero-base-blimp', { waitUntil: 'load' });
  const schematicLoadTime = Date.now() - startSchematic;
  console.log(`Schematic page load: ${schematicLoadTime}ms`);

  await page.waitForTimeout(2000);
  const schematicPerf = await getPagePerf(page);
  printPerf('Schematic Page', schematicPerf);

  // 3. Navigate back to homepage (simulates clicking logo - full navigation)
  console.log('\n=== Step 3: Return to homepage (warm) ===');
  const startHome = Date.now();
  await page.goto(BASE + '/', { waitUntil: 'load' });
  const homeLoadTime = Date.now() - startHome;
  console.log(`Homepage return load: ${homeLoadTime}ms`);

  await page.waitForTimeout(4000);
  const returnPerf = await getPagePerf(page);
  printPerf('Homepage Return', returnPerf);

  // Assertions
  expect(returnPerf.navigation.ttfb).toBeLessThan(1000);
  expect(homeLoadTime).toBeLessThan(10000);
});

async function getPagePerf(page: any) {
  return page.evaluate(() => {
    const nav = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming;
    const entries = performance.getEntriesByType('resource') as PerformanceResourceTiming[];

    const images = entries.filter(e => e.name.includes('/api/files/'));

    return {
      navigation: {
        ttfb: Math.round(nav.responseStart - nav.startTime),
        domInteractive: Math.round(nav.domInteractive - nav.startTime),
        domContentLoaded: Math.round(nav.domContentLoadedEventEnd - nav.startTime),
        loadEvent: Math.round(nav.loadEventEnd - nav.startTime),
        transferSizeKB: Math.round((nav.transferSize || 0) / 1024),
      },
      images: images.map(e => ({
        url: e.name.replace(/https?:\/\/[^/]+/, '').substring(0, 90),
        duration: Math.round(e.duration),
        transferSize: e.transferSize,
        startTime: Math.round(e.startTime),
      })).sort((a, b) => b.duration - a.duration),
      summary: {
        totalResources: entries.length,
        totalTransferKB: Math.round(entries.reduce((s, e) => s + (e.transferSize || 0), 0) / 1024),
        imageCount: images.length,
        imageTotalKB: Math.round(images.reduce((s, e) => s + (e.transferSize || 0), 0) / 1024),
      },
    };
  });
}

function printPerf(label: string, perf: any) {
  console.log(`\n--- ${label} - Navigation ---`);
  console.log(`  TTFB: ${perf.navigation.ttfb}ms`);
  console.log(`  DOM Interactive: ${perf.navigation.domInteractive}ms`);
  console.log(`  DOMContentLoaded: ${perf.navigation.domContentLoaded}ms`);
  console.log(`  Load Event: ${perf.navigation.loadEvent}ms`);
  console.log(`  HTML size: ${perf.navigation.transferSizeKB}KB`);

  console.log(`\n--- ${label} - Resources ---`);
  console.log(`  Total: ${perf.summary.totalResources} resources (${perf.summary.totalTransferKB}KB)`);
  console.log(`  Images: ${perf.summary.imageCount} (${perf.summary.imageTotalKB}KB)`);

  if (perf.images.length > 0) {
    const durations = perf.images.map((i: any) => i.duration);
    const sorted = [...durations].sort((a: number, b: number) => a - b);
    const avg = Math.round(sorted.reduce((a: number, b: number) => a + b, 0) / sorted.length);
    const p50 = sorted[Math.floor(sorted.length * 0.5)];
    const p95 = sorted[Math.floor(sorted.length * 0.95)] || sorted[sorted.length - 1];

    console.log(`\n--- ${label} - Image Timing ---`);
    console.log(`  Count: ${durations.length}`);
    console.log(`  Avg: ${avg}ms | P50: ${p50}ms | P95: ${p95}ms | Max: ${sorted[sorted.length - 1]}ms`);
    console.log('  Top 10 slowest:');
    perf.images.slice(0, 10).forEach((img: any) => {
      console.log(`    ${img.duration}ms | ${Math.round(img.transferSize / 1024)}KB | start:${img.startTime}ms | ${img.url}`);
    });
  }
}
