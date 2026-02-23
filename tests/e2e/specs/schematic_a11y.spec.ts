import { test, expect } from '@playwright/test';
import { AxeBuilder } from '@axe-core/playwright';

// Runs axe-core accessibility scan on the seeded schematic page.
// Requires the "e2e-test-schematic" record from global-setup.ts.

test.describe('schematic page accessibility', () => {
  test('has no serious a11y violations', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';

    const resp = await page.goto(url + '/schematics/e2e-test-schematic');

    // If the schematic page 404s (e.g. not moderated), skip gracefully
    if (resp?.status() === 404) {
      test.skip(true, 'schematic page returned 404 — seed data may not be moderated');
      return;
    }

    expect(resp?.status()).toBeLessThan(500);

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();

    // Filter to only serious and critical violations
    const serious = (accessibilityScanResults.violations || []).filter(v =>
      ['serious', 'critical'].includes(v.impact || ''),
    );

    expect(serious, 'no serious/critical a11y violations expected').toHaveLength(0);
  });
});
