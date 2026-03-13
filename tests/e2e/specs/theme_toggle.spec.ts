import { test, expect } from '@playwright/test';

test.describe('Theme toggle', () => {
  test('window.setTheme is defined after page load', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/');

    // theme.js should expose window.setTheme
    const hasSetTheme = await page.evaluate(() => typeof (window as any).setTheme === 'function');
    expect(hasSetTheme, 'window.setTheme should be a function').toBe(true);
  });

  test('clicking light mode button switches to light theme', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/');

    // Default theme is dark — the "switch to light" button should be visible
    const lightBtn = page.locator('button.hide-theme-light');
    // The light button has class hide-theme-light, which is shown when theme is dark
    // (hide-theme-light means "hide this when theme IS light")
    await expect(lightBtn).toBeAttached();

    // Click the light mode toggle
    await lightBtn.click();

    // After clicking, data-bs-theme should be "light"
    const theme = await page.evaluate(() => document.documentElement.getAttribute('data-bs-theme'));
    expect(theme).toBe('light');

    // localStorage should persist the choice
    const stored = await page.evaluate(() => localStorage.getItem('createmodTheme'));
    expect(stored).toBe('light');
  });

  test('clicking dark mode button switches back to dark theme', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/');

    // First switch to light
    await page.evaluate(() => (window as any).setTheme('light'));
    const lightTheme = await page.evaluate(() => document.documentElement.getAttribute('data-bs-theme'));
    expect(lightTheme).toBe('light');

    // The dark-mode button (hide-theme-dark) should now be visible
    const darkBtn = page.locator('button.hide-theme-dark');
    await expect(darkBtn).toBeAttached();
    await darkBtn.click();

    // After clicking, theme should be dark again
    const theme = await page.evaluate(() => document.documentElement.getAttribute('data-bs-theme'));
    expect(theme).toBe('dark');

    const stored = await page.evaluate(() => localStorage.getItem('createmodTheme'));
    expect(stored).toBe('dark');
  });

  test('theme persists across page navigation', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/');

    // Switch to light
    await page.evaluate(() => (window as any).setTheme('light'));

    // Navigate to another page
    await page.goto(url + '/rules');

    // Theme should still be light
    const theme = await page.evaluate(() => document.documentElement.getAttribute('data-bs-theme'));
    expect(theme).toBe('light');
  });

  test('toggle buttons have correct visibility for dark theme', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/');

    // In dark mode: hide-theme-dark buttons should be hidden, hide-theme-light should be visible
    await page.evaluate(() => (window as any).setTheme('dark'));

    // The "switch to light" button (class hide-theme-light) should be visible in dark mode
    const lightBtn = page.locator('button.hide-theme-light');
    await expect(lightBtn).toBeAttached();
    const lightDisplay = await lightBtn.evaluate(el => getComputedStyle(el).display);
    expect(lightDisplay).not.toBe('none');

    // The "switch to dark" button (class hide-theme-dark) should be hidden in dark mode
    const darkBtn = page.locator('button.hide-theme-dark');
    await expect(darkBtn).toBeAttached();
    const darkDisplay = await darkBtn.evaluate(el => el.style.display);
    expect(darkDisplay).toBe('none');
  });
});
