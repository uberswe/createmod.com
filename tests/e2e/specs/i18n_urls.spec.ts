import { test, expect } from '@playwright/test';

const LANG_PREFIXES = [
  { lang: 'de', prefix: '/de', name: 'German' },
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
    // 8 languages + x-default = 9
    expect(hreflangs.length).toBeGreaterThanOrEqual(9);

    // Check x-default exists
    const xDefault = await page.locator('link[rel="alternate"][hreflang="x-default"]').getAttribute('href');
    expect(xDefault).toContain('createmod.com');
  });

  test('German page has hreflang tags', async ({ page }) => {
    await page.goto('/de/');
    const hreflangs = await page.locator('link[rel="alternate"][hreflang]').all();
    expect(hreflangs.length).toBeGreaterThanOrEqual(9);
  });
});

test.describe('language switcher', () => {
  test('language switcher links use subdirectory URLs', async ({ page }) => {
    await page.goto('/');
    // Open language dropdown
    const langTrigger = page.locator('.lang-flag');
    await langTrigger.click();

    // Check German link points to /de/
    const deLink = page.locator('.dropdown-item >> text=Deutsch');
    const deHref = await deLink.getAttribute('href');
    expect(deHref).toContain('/de');

    // Check Russian link
    const ruLink = page.locator('.dropdown-item >> text=Русский');
    const ruHref = await ruLink.getAttribute('href');
    expect(ruHref).toContain('/ru');
  });

  test('language switcher on German page preserves path', async ({ page }) => {
    await page.goto('/de/schematics');
    const langTrigger = page.locator('.lang-flag');
    await langTrigger.click();

    // English link should go to /schematics (no prefix)
    const enLink = page.locator('.dropdown-item >> text=English');
    const enHref = await enLink.getAttribute('href');
    expect(enHref).toBe('/schematics');
  });
});

test.describe('navigation links are prefixed', () => {
  test('sidebar links are prefixed on German page', async ({ page }) => {
    await page.goto('/de/');
    // Check sidebar nav links contain /de/ prefix
    const sidebarLinks = await page.locator('.sidebar-nav a.sidebar-item').all();
    if (sidebarLinks.length === 0) {
      test.skip(true, 'no sidebar nav links found — layout may differ in CI');
      return;
    }
    for (const link of sidebarLinks) {
      const href = await link.getAttribute('href');
      expect(href, `sidebar link should start with /de`).toMatch(/^\/de(\/|$)/);
    }
  });

  test('sidebar links have no prefix on English page', async ({ page }) => {
    await page.goto('/');
    const sidebarLinks = await page.locator('.sidebar-nav a.sidebar-item').all();
    if (sidebarLinks.length === 0) {
      test.skip(true, 'no sidebar nav links found — layout may differ in CI');
      return;
    }
    for (const link of sidebarLinks) {
      const href = await link.getAttribute('href');
      // English links should NOT start with a language prefix
      expect(href).toMatch(/^\//);
      expect(href).not.toMatch(/^\/(de|es|pl|pt-br|pt-pt|ru|zh)\//);
    }
  });
});

test.describe('HTMX interceptor', () => {
  test('clicking a link on German page stays in German prefix', async ({ page }) => {
    await page.goto('/de/');
    // Click a navigation link via HTMX boost (e.g. search in sidebar)
    await page.locator('.sidebar-nav a[href="/de/search"]').click();
    await page.waitForURL(/\/de\/search/);
    expect(page.url()).toContain('/de/search');

    // Verify the page still shows German language
    const htmlLang = await page.locator('html').getAttribute('lang');
    expect(htmlLang).toBe('de');
  });
});
