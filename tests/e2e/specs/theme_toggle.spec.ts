import { test, expect } from '@playwright/test';

test.describe('Theme toggle', () => {
  test('window.setTheme is defined after page load', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/');

    // theme.js should expose window.setTheme
    const hasSetTheme = await page.evaluate(() => typeof (window as any).setTheme === 'function');
    expect(hasSetTheme, 'window.setTheme should be a function').toBe(true);
  });

  test('clicking theme toggle switches to light theme', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/');

    // Default theme is dark — click the single toggle button to switch to light
    const toggleBtn = page.locator('#theme-toggle');
    await expect(toggleBtn).toBeAttached();
    await toggleBtn.click();

    // After clicking, data-cm-theme should be "light"
    const theme = await page.evaluate(() => document.documentElement.getAttribute('data-cm-theme'));
    expect(theme).toBe('light');

    // localStorage should persist the choice
    const stored = await page.evaluate(() => localStorage.getItem('createmodTheme'));
    expect(stored).toBe('light');
  });

  test('clicking theme toggle twice switches back to dark theme', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/');

    // First switch to light
    await page.evaluate(() => (window as any).setTheme('light'));
    const lightTheme = await page.evaluate(() => document.documentElement.getAttribute('data-cm-theme'));
    expect(lightTheme).toBe('light');

    // Click toggle to go back to dark
    const toggleBtn = page.locator('#theme-toggle');
    await expect(toggleBtn).toBeAttached();
    await toggleBtn.click();

    // After clicking, theme should be dark again
    const theme = await page.evaluate(() => document.documentElement.getAttribute('data-cm-theme'));
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
    const theme = await page.evaluate(() => document.documentElement.getAttribute('data-cm-theme'));
    expect(theme).toBe('light');
  });

  test('theme icons have correct visibility', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/');

    // In dark mode: moon icon (.theme-icon-dark) hidden, sun icon (.theme-icon-light) visible
    await page.evaluate(() => (window as any).setTheme('dark'));

    const toggleBtn = page.locator('#theme-toggle');
    await expect(toggleBtn).toBeAttached();

    const moonDisplay = await toggleBtn.locator('.theme-icon-dark').evaluate(el => getComputedStyle(el).display);
    expect(moonDisplay).toBe('none');

    const sunDisplay = await toggleBtn.locator('.theme-icon-light').evaluate(el => getComputedStyle(el).display);
    expect(sunDisplay).not.toBe('none');

    // Switch to light mode: moon visible, sun hidden
    await page.evaluate(() => (window as any).setTheme('light'));

    const moonDisplayLight = await toggleBtn.locator('.theme-icon-dark').evaluate(el => getComputedStyle(el).display);
    expect(moonDisplayLight).not.toBe('none');

    const sunDisplayLight = await toggleBtn.locator('.theme-icon-light').evaluate(el => getComputedStyle(el).display);
    expect(sunDisplayLight).toBe('none');
  });
});
