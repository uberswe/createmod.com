import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  globalSetup: './tests/e2e/global-setup.ts',
  testDir: './tests/e2e',
  timeout: 60_000,
  retries: 1,
  use: {
    baseURL: process.env.APP_BASE_URL || 'http://localhost:8080',
    trace: 'retain-on-failure',
    video: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],
  workers: 3,
});
