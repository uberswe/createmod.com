import { test, expect } from '@playwright/test';
import { TEST_USER_EMAIL, TEST_USER_PASSWORD, loginViaCookie } from '../helpers/auth';

test.describe('login flow', () => {
  test('login form submits and header updates to show username', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';

    // Navigate to login page
    await page.goto(url + '/login');

    // Verify login form is visible
    await expect(page.locator('#login-form')).toBeVisible();
    await expect(page.locator('input[name="username"]')).toBeVisible();
    await expect(page.locator('input[name="password"]')).toBeVisible();

    // Fill in credentials and submit
    await page.fill('input[name="username"]', TEST_USER_EMAIL);
    await page.fill('input[name="password"]', TEST_USER_PASSWORD);

    // Use Promise.all to submit and wait for navigation simultaneously
    await Promise.all([
      page.waitForURL(url + '/', { timeout: 15000 }),
      page.click('button[type="submit"]'),
    ]);

    // Verify the auth cookie is set
    const cookies = await page.context().cookies();
    const authCookie = cookies.find(c => c.name === 'create-mod-auth');
    expect(authCookie, 'auth cookie should be set after login').toBeTruthy();
    expect(authCookie!.value).not.toBe('');

    // Verify the header shows authenticated state (username, not "Login" button)
    await expect(page.locator('#login-button')).not.toBeVisible();
    await expect(page.locator('.auth-section')).toBeVisible();
  });

  test('login with invalid credentials stays on login page', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';

    await page.goto(url + '/login');
    await page.fill('input[name="username"]', 'nonexistent@example.com');
    await page.fill('input[name="password"]', 'wrongpassword');
    await page.click('button[type="submit"]');

    // Should redirect back to /login (or stay on login page)
    await page.waitForURL('**/login**', { timeout: 10000 });

    // Auth cookie should NOT be set
    const cookies = await page.context().cookies();
    const authCookie = cookies.find(c => c.name === 'create-mod-auth');
    expect(authCookie?.value ?? '').toBe('');
  });

  test('login form bypasses HTMX boost for proper cookie handling', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/login');

    // The login form should have hx-boost="false" to bypass HTMX interception.
    // This ensures Set-Cookie headers from the server are processed by the browser,
    // since XHR/fetch responses silently ignore Set-Cookie per the fetch spec.
    const form = page.locator('#login-form');
    await expect(form).toHaveAttribute('hx-boost', 'false');
  });

  test('authenticated user sees avatar and profile dropdown in header', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';

    // Navigate first so the cookie domain is valid for the browser context
    await page.goto(url + '/');
    await loginViaCookie(page, url);

    // Navigate to homepage (with cookie now set)
    await page.goto(url + '/');

    // Should see auth section (avatar + dropdown), not login button
    await expect(page.locator('#login-button')).not.toBeVisible();
    await expect(page.locator('.auth-section')).toBeVisible({ timeout: 10000 });

    // Click the avatar dropdown to reveal menu
    await page.locator('.auth-section [data-bs-toggle="dropdown"]').click();

    // The dropdown menu should become visible after clicking
    const dropdownMenu = page.locator('.auth-section .dropdown-menu');
    await expect(dropdownMenu).toBeVisible({ timeout: 5000 });

    // Should have profile, settings, and logout links in the dropdown
    await expect(dropdownMenu.locator('a[href="/profile"]')).toBeVisible();
    await expect(dropdownMenu.locator('a[href="/settings"]')).toBeVisible();
    await expect(dropdownMenu.locator('a[href="/logout"]')).toBeVisible();
  });
});

test.describe('signup flow', () => {
  test('register page renders signup form', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/register');

    // Verify the signup form is present
    await expect(page.locator('#signup-form')).toBeVisible();
    await expect(page.locator('#username')).toBeVisible();
    await expect(page.locator('#email')).toBeVisible();
    await expect(page.locator('#password')).toBeVisible();
    await expect(page.locator('#terms')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test('signup form validates empty fields', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/register');

    // Submit with empty fields
    await page.click('button[type="submit"]');

    // Fields should get is-invalid class
    await expect(page.locator('#username.is-invalid')).toBeVisible();
    await expect(page.locator('#password.is-invalid')).toBeVisible();
    await expect(page.locator('#email.is-invalid')).toBeVisible();
  });

  test('signup form validates terms checkbox', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/register');

    // Fill all fields but don't check terms
    await page.fill('#username', 'testvalidation');
    await page.fill('#email', 'testvalidation@example.com');
    await page.fill('#password', 'TestPassword123!');
    await page.click('button[type="submit"]');

    // Terms checkbox should be marked invalid
    await expect(page.locator('#terms.is-invalid')).toBeVisible();
  });

  test('register page has links to login and terms', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/register');

    // Link to login
    await expect(page.locator('a[href="/login"]')).toBeVisible();

    // Link to terms of service (use .first() as footer may also contain one)
    await expect(page.locator('a[href="/terms-of-service"]').first()).toBeVisible();
  });
});
