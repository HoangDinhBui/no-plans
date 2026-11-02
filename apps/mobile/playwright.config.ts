import { defineConfig } from '@playwright/test';
export default defineConfig({
  testDir: './e2e', timeout: 45000, workers: 1,
  use: { baseURL: 'http://127.0.0.1:8092', viewport: { width: 390, height: 844 }, trace: 'retain-on-failure', screenshot: 'only-on-failure' },
  webServer: {
    command: 'node node_modules/expo/bin/cli start --web --localhost --port 8092', url: 'http://127.0.0.1:8092', timeout: 180000, reuseExistingServer: false,
    env: { CI: '1', EXPO_PUBLIC_API_URL: 'https://api.no-plans.test', EXPO_PUBLIC_SUPABASE_URL: 'https://auth.no-plans.test', EXPO_PUBLIC_SUPABASE_PUBLISHABLE_KEY: 'public-fixture-key', EXPO_PUBLIC_INVITE_WEB_URL: 'https://invite.no-plans.test' },
  },
});
