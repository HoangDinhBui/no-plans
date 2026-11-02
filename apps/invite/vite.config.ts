import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: { port: 5173, strictPort: true, proxy: { '/v1': 'http://127.0.0.1:8080' }, headers: { 'Referrer-Policy': 'no-referrer', 'X-Content-Type-Options': 'nosniff' } },
  preview: { headers: { 'Referrer-Policy': 'no-referrer' } },
  test: { environment: 'jsdom', include: ['src/**/*.test.{ts,tsx}'], restoreMocks: true, setupFiles: ['src/test-setup.ts'] },
});
