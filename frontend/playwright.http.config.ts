import { defineConfig, devices } from '@playwright/test'
import { OPEN_URL, TOKEN_URL } from './e2e-http/servers'

// Runs against the real headless server (cmd/server), no Wails mock and no
// page.route. retries stays 0 so a flaky real server is never masked.
export default defineConfig({
  testDir: './e2e-http',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  globalTeardown: './e2e-http/global-teardown.mjs',
  reporter: [['list'], ['html', { outputFolder: '../playwright-report-http', open: 'never' }]],
  use: {
    baseURL: OPEN_URL,
    locale: 'en-US',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    actionTimeout: 8000,
    navigationTimeout: 20000,
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: [
    {
      command: 'node e2e-http/start-server.mjs open',
      url: `${OPEN_URL}/api/health`,
      reuseExistingServer: false,
      gracefulShutdown: { signal: 'SIGTERM', timeout: 8000 },
      timeout: 60000,
      stdout: 'pipe',
      stderr: 'pipe',
    },
    {
      command: 'node e2e-http/start-server.mjs token',
      url: `${TOKEN_URL}/api/health`,
      reuseExistingServer: false,
      gracefulShutdown: { signal: 'SIGTERM', timeout: 8000 },
      timeout: 60000,
      stdout: 'pipe',
      stderr: 'pipe',
    },
  ],
})
