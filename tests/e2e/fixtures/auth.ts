import { test as base } from '@playwright/test';
import { authenticateUser } from '../helpers/auth';

// Extend the base test with an authenticated page fixture.
export const test = base.extend<{ userPage: any }>({
  userPage: async ({ browser }, use) => {
    const context = await browser.newContext();
    const page = await context.newPage();

    // Fast login via the app's /login endpoint
    try {
      const token = await authenticateUser();
      await context.addCookies([
        {
          name: 'create-mod-auth',
          value: token,
          domain: 'localhost',
          path: '/',
          httpOnly: true,
        },
      ]);
    } catch (e) {
      // If the test user isn't seeded yet, tests that require auth should skip or handle 401s.
    }

    await use(page);
    await context.close();
  },
});

export const expect = base.expect;
