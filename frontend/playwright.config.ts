import { defineConfig, devices } from '@playwright/test';
export default defineConfig({
 testDir: './tests/e2e', fullyParallel: false, workers: 1, timeout: 60000,
 expect: { timeout: 20000 }, retries: process.env.CI ? 1 : 0,
 reporter: [['list'], ['html', { open: 'never' }]],
 use: { baseURL: process.env.E2E_BASE_URL || 'http://localhost:8080', trace: 'retain-on-failure', screenshot: 'only-on-failure', viewport: { width: 1440, height: 1000 } },
 projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 1000 }, launchOptions: {args:['--enable-unsafe-swiftshader']} } }],
});
