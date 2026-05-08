import { test, expect } from '@playwright/test';

test.describe('Search page filters', () => {
  test('Best match is the default selected sort', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/search');

    // The sort dropdown should default to "Best match" (value="1")
    const sortSelect = page.locator('#advanced-search-form select[name="sort"]');
    await expect(sortSelect).toHaveValue('1');
  });

  test('Changing sort filter updates search results via HTMX', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/search');

    // Wait for initial results to load
    await page.waitForSelector('#search-results');

    // Select "Newest" from the sort dropdown
    const sortSelect = page.locator('#advanced-search-form select[name="sort"]');
    await sortSelect.selectOption('2');

    // Wait for HTMX to update the results
    await page.waitForTimeout(1000);

    // URL should have been updated with sort param
    expect(page.url()).toContain('sort=2');
  });

  test('Changing category filter updates search results', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/search');

    await page.waitForSelector('#search-results');

    // Change category dropdown
    const categorySelect = page.locator('#advanced-search-form select[name="category"]');
    const options = await categorySelect.locator('option').all();

    // If there are categories beyond "All", select the second one
    if (options.length > 1) {
      const value = await options[1].getAttribute('value');
      if (value) {
        await categorySelect.selectOption(value);
        await page.waitForTimeout(1000);
        expect(page.url()).toContain(`category=${value}`);
      }
    }
  });

  test('Typing in search hero input updates results', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/search');

    await page.waitForSelector('#search-results');

    // Type a search term character by character (triggers keyup events for HTMX)
    const heroInput = page.locator('#search-hero-input');
    await heroInput.click();
    await heroInput.pressSequentially('farm', { delay: 50 });

    // Wait for HTMX debounce (800ms) + response
    await page.waitForTimeout(3000);

    // URL should reflect the search
    expect(page.url()).toContain('q=farm');
  });

  test('Changing rating filter updates results', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';
    await page.goto(url + '/search');

    await page.waitForSelector('#search-results');

    // Move the rating range slider to 4 stars
    const slider = page.locator('#rating-slider');
    await slider.fill('4');
    await slider.dispatchEvent('input');

    await page.waitForTimeout(1000);
    expect(page.url()).toContain('rating=4');
  });
});
