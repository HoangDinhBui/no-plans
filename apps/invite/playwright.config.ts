import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './e2e', fullyParallel: false, workers: 2, timeout: 30000,
  use: { baseURL: 'http://127.0.0.1:5175', trace: 'retain-on-failure', screenshot: 'only-on-failure' },
  webServer: {
    command: 'node node_modules/vite/bin/vite.js --host 127.0.0.1 --port 5175', url: 'http://127.0.0.1:5175', reuseExistingServer: false,
    env: { VITE_SUPABASE_URL: 'https://auth.no-plans.test', VITE_SUPABASE_ANON_KEY: 'public-test-key' },
  },
  projects: [
    { name: 'desktop', use: { browserName: 'chromium', viewport: { width: 1440, height: 1000 }, colorScheme: 'light' } },
    { name: 'mobile', use: { browserName: 'chromium', viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, colorScheme: 'light' } },
    { name: 'dark', use: { browserName: 'chromium', viewport: { width: 1440, height: 1000 }, colorScheme: 'dark' } },
  ],
});
