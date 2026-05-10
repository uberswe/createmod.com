import { test, expect } from '@playwright/test';

const LANG_PREFIXES = [
  { lang: 'es', prefix: '/es', name: 'Spanish' },
  { lang: 'pl', prefix: '/pl', name: 'Polish' },
  { lang: 'pt-BR', prefix: '/pt-br', name: 'Portuguese (Brazil)' },
  { lang: 'pt-PT', prefix: '/pt-pt', name: 'Portuguese (Portugal)' },
  { lang: 'ru', prefix: '/ru', name: 'Russian' },
  { lang: 'zh-Hans', prefix: '/zh', name: 'Chinese (Simplified)' },
];

const PAGES = ['/', '/schematics', '/upload', '/search', '/guides', '/videos', '/collections', '/mods', '/explore', '/contact', '/rules', '/login'];

test.describe('i18n URL routing', () => {
  test('English root pages return 200', async ({ page }) => {
    for (const p of PAGES) {
      const resp = await page.goto(p);
      expect(resp?.status(), `GET ${p}`).toBe(200);
    }
  });

  for (const { lang, prefix, name } of LANG_PREFIXES) {
    test(`${name} (${prefix}) homepage loads`, async ({ page }) => {
      const resp = await page.goto(prefix + '/');
      expect(resp?.status(), `GET ${prefix}/`).toBe(200);
      const htmlLang = await page.locator('html').getAttribute('lang');
      expect(htmlLang).toBe(lang);
      const dataPrefix = await page.locator('html').getAttribute('data-lang-prefix');
      expect(dataPrefix).toBe(prefix.slice(1)); // strip leading /
    });

    test(`${name} (${prefix}) bare prefix redirects to trailing slash`, async ({ page }) => {
      // Playwright follows redirects, so check final URL
      await page.goto(prefix);
      expect(page.url()).toContain(prefix + '/');
    });

    test(`${name} (${prefix}) key pages return 200`, async ({ page }) => {
      const testPages = ['/schematics', '/search', '/upload', '/login'];
      for (const p of testPages) {
        const resp = await page.goto(prefix + p);
        expect(resp?.status(), `GET ${prefix}${p}`).toBe(200);
      }
    });
  }
});

test.describe('hreflang tags', () => {
  test('homepage has hreflang tags for all languages', async ({ page }) => {
    await page.goto('/');
    const hreflangs = await page.locator('link[rel="alternate"][hreflang]').all();
    // 7 languages + x-default = 8
    expect(hreflangs.length).toBeGreaterThanOrEqual(8);

    // Check x-default exists
    const xDefault = await page.locator('link[rel="alternate"][hreflang="x-default"]').getAttribute('href');
    expect(xDefault).toContain('createmod.com');
  });

  test('Spanish page has hreflang tags', async ({ page }) => {
    await page.goto('/es/');
    const hreflangs = await page.locator('link[rel="alternate"][hreflang]').all();
    expect(hreflangs.length).toBeGreaterThanOrEqual(8);
  });
});

test.describe('language switcher', () => {
  test('language switcher links use /lang endpoint to set cookie', async ({ page }) => {
    await page.goto('/');
    // Open language dropdown (globe icon trigger)
    const langTrigger = page.locator('a[aria-label="Select language"]');
    await langTrigger.click();

    // Check Spanish link routes through /lang endpoint
    const esLink = page.locator('.dropdown-item >> text=Español');
    const esHref = await esLink.getAttribute('href');
    expect(esHref).toContain('/lang?l=es');

    // Check Russian link
    const ruLink = page.locator('.dropdown-item >> text=Русский');
    const ruHref = await ruLink.getAttribute('href');
    expect(ruHref).toContain('/lang?l=ru');
  });

  test('language switcher on Spanish page preserves path', async ({ page }) => {
    await page.goto('/es/schematics');
    const langTrigger = page.locator('a[aria-label="Select language"]');
    await langTrigger.click();

    // English link should route through /lang with return_to
    const enLink = page.locator('.dropdown-item >> text=English');
    const enHref = await enLink.getAttribute('href');
    expect(enHref).toContain('/lang?l=en');
    expect(enHref).toContain('return_to=');
  });
});

