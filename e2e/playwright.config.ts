import { defineConfig, devices } from '@playwright/test';

// The built site (nginx) and the backend from docker-compose.e2e.yml. 127.0.0.1, not localhost:
// localhost can resolve to ::1 first, which is not where docker publishes the ports.
export const WEB_URL = process.env.WEB_URL ?? 'http://127.0.0.1:6190';
export const BACKEND_URL = process.env.BACKEND_URL ?? 'http://127.0.0.1:6191';

// CI builds both images from scratch.
const STACK_TIMEOUT = process.env.CI ? 600_000 : 240_000;

export default defineConfig({
  testDir: './tests',
  // The backend keeps no state that tests share (every test has its own rate-limit client,
  // see tests/fixtures.ts), so tests can run in parallel.
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }], ['list']] : [['html', { open: 'never' }], ['list']],
  use: {
    baseURL: WEB_URL,
    locale: 'nl-NL',
    timezoneId: 'Europe/Amsterdam',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    { name: 'desktop-chromium', use: { ...devices['Desktop Chrome'] } },
    { name: 'android-chromium', use: { ...devices['Pixel 7'] } },
    { name: 'iphone-webkit', use: { ...devices['iPhone 15'] } },
  ],
  // Boot the stack before the run. Set E2E_NO_WEBSERVER=1 when it already runs (npm run stack:up).
  webServer: process.env.E2E_NO_WEBSERVER
    ? undefined
    : [
        {
          command: 'docker compose -f ../docker-compose.e2e.yml up --build',
          url: `${BACKEND_URL}/healthz`,
          reuseExistingServer: !process.env.CI,
          timeout: STACK_TIMEOUT,
          // SIGTERM, not a kill, so docker compose stops the containers it started.
          gracefulShutdown: { signal: 'SIGTERM', timeout: 20_000 },
        },
        {
          // nginx comes up after the backend; wait until it serves the site too.
          command: 'node -e "setInterval(() => {}, 1 << 30)"',
          url: WEB_URL,
          reuseExistingServer: true,
          timeout: STACK_TIMEOUT,
        },
      ],
});
