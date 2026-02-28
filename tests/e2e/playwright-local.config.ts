import { defineConfig, devices } from '@playwright/test';

// Minimal config for running individual tests against a local server.
// No global setup (seeding) — assumes the server is already running with data.
export default defineConfig({
  testDir: './specs',
  timeout: 60_000,
  retries: 0,
  use: {
    baseURL: process.env.APP_BASE_URL || 'http://localhost:8090',
    trace: 'off',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],
  workers: 1,
});
