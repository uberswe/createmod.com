import { test as base } from '@playwright/test';
import PocketBase from 'pocketbase';

// Extend the base test with an authenticated page fixture.
// Adjust cookie name and auth flow to match the app when backend is wired.
export const test = base.extend<{ userPage: any }>({
  userPage: async ({ browser }, use) => {
    const context = await browser.newContext();
    const page = await context.newPage();

    // Optional fast login via PocketBase REST
    try {
      const pb = new PocketBase(process.env.PB_URL ?? 'http://localhost:8090');
      const auth = await pb.collection('users').authWithPassword('user@example.com', 'password123');
      await context.addCookies([
        {
          name: 'create-mod-auth', // replace with actual auth cookie if different
          value: auth.token,
          domain: 'localhost',
          path: '/',
          httpOnly: true,
        },
      ]);
    } catch (e) {
      // If PB isn't seeded yet, tests that require auth should skip or handle 401s.
      // For the smoke test we don't rely on auth.
    }

    await use(page);
    await context.close();
  },
});

export const expect = base.expect;
