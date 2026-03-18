import { test, expect } from '@playwright/test';

// Verify that clicking a schematic image opens FSLightbox instead of navigating
// to the raw image URL — both on direct page load and after HTMX boost navigation.

test.describe('Schematic image lightbox', () => {
  test('clicking the featured image opens a lightbox overlay on direct load', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';

    // Navigate to the schematics listing and find a schematic link
    await page.goto(url + '/schematics');
    const firstSchematic = page.locator('a[href^="/schematics/"]').first();
    await expect(firstSchematic).toBeAttached({ timeout: 10_000 });
    const href = await firstSchematic.getAttribute('href');
    expect(href).toBeTruthy();

    // Go directly to the schematic page (full page load)
    await page.goto(url + href!);
    await page.waitForLoadState('networkidle');

    // The featured image link should have data-fslightbox="gallery"
    const imageLink = page.locator('a[data-fslightbox="gallery"]').first();
    await expect(imageLink).toBeAttached({ timeout: 10_000 });

    // Wait for FSLightbox to load and initialize
    await page.waitForFunction(() => typeof (window as any).refreshFsLightbox === 'function', null, { timeout: 15_000 });

    // Record the current URL before clicking
    const urlBefore = page.url();

    // Click the featured image
    await imageLink.click();

    // FSLightbox creates a container with class "fslightbox-container" when open
    const lightboxContainer = page.locator('.fslightbox-container');
    await expect(lightboxContainer).toBeVisible({ timeout: 5_000 });

    // The page URL should NOT have changed (no navigation to the raw image)
    expect(page.url()).toBe(urlBefore);
  });

  test('lightbox opens after HTMX navigation from one schematic to another', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';

    // Find two distinct schematic hrefs from the listing
    await page.goto(url + '/schematics');
    await page.waitForLoadState('networkidle');
    const schematicLinks = page.locator('a[href^="/schematics/"]');
    const count = await schematicLinks.count();
    if (count < 2) {
      test.skip(true, 'Need at least 2 schematics');
      return;
    }
    const href1 = await schematicLinks.nth(0).getAttribute('href');
    const href2 = await schematicLinks.nth(1).getAttribute('href');
    if (!href1 || !href2 || href1 === href2) {
      test.skip(true, 'Could not find two distinct schematic links');
      return;
    }

    // Full-page load of the first schematic — FSLightbox loads here for the
    // first time and auto-initializes.
    await page.goto(url + href1);
    await page.waitForLoadState('networkidle');
    await page.waitForFunction(() => typeof (window as any).refreshFsLightbox === 'function', null, { timeout: 15_000 });

    // Now navigate to the second schematic via HTMX boost by clicking a link.
    // Use the browser back button to the listing, then click the 2nd link —
    // this keeps us in the same HTMX session.
    // Instead, we can directly evaluate a boosted navigation to the 2nd href:
    await page.evaluate((h) => {
      // Simulate an HTMX-boosted navigation by clicking a temporary link
      const a = document.createElement('a');
      a.href = h;
      a.textContent = 'temp';
      document.body.appendChild(a);
      (window as any).htmx.process(a);  // make HTMX aware of the new element
      a.click();
    }, href2);

    // Wait for HTMX to swap in the new schematic page
    await page.waitForLoadState('networkidle');

    // The new schematic page should have gallery image links
    const imageLink = page.locator('a[data-fslightbox="gallery"]').first();
    await expect(imageLink).toBeAttached({ timeout: 10_000 });

    // Wait a moment for FSLightbox reinit (the fix calls refreshFsLightbox)
    await page.waitForTimeout(1000);

    // Record URL before clicking the image
    const urlBefore = page.url();

    // Click the featured image on the SECOND schematic page
    await imageLink.click();

    // The lightbox should open — this is the critical assertion.
    // Without the fix, FSLightbox never re-scans the new DOM, so the click
    // falls through to the <a> href and navigates to the raw image URL.
    const lightboxContainer = page.locator('.fslightbox-container');
    await expect(lightboxContainer).toBeVisible({ timeout: 5_000 });

    // URL must not have changed
    expect(page.url()).toBe(urlBefore);
  });

  test('lightbox has navigation arrows for gallery with multiple images', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:8080';

    // Find a schematic with a gallery (multiple images)
    await page.goto(url + '/schematics');
    const schematicLink = page.locator('a[href^="/schematics/"]').first();
    await expect(schematicLink).toBeAttached({ timeout: 10_000 });
    const href = await schematicLink.getAttribute('href');
    await page.goto(url + href!);
    await page.waitForLoadState('networkidle');

    // Check if this schematic has gallery images
    const galleryLinks = page.locator('a[data-fslightbox="gallery"]');
    const galleryCount = await galleryLinks.count();

    if (galleryCount < 2) {
      test.skip(true, 'Need a schematic with multiple images to test navigation arrows');
      return;
    }

    // Wait for FSLightbox
    await page.waitForFunction(() => typeof (window as any).refreshFsLightbox === 'function', null, { timeout: 15_000 });

    // Click the first gallery image
    await galleryLinks.first().click();

    // Lightbox should be open
    const lightboxContainer = page.locator('.fslightbox-container');
    await expect(lightboxContainer).toBeVisible({ timeout: 5_000 });

    // Should have navigation arrows (next/prev buttons)
    const navButtons = page.locator('.fslightbox-slide-btn');
    await expect(navButtons.first()).toBeVisible({ timeout: 3_000 });
  });
});
