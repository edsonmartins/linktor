import { defineConfig, devices } from '@playwright/test'

const baseURL = process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:3001'
const apiURL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081/api/v1'
const wsURL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8081/api/v1/ws'
const webhookBaseURL = process.env.NEXT_PUBLIC_WEBHOOK_BASE_URL || 'http://localhost:8081'

export default defineConfig({
  testDir: './e2e-real',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: 'html',
  use: {
    baseURL,
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: `NEXT_PUBLIC_API_URL=${apiURL} NEXT_PUBLIC_WS_URL=${wsURL} NEXT_PUBLIC_WEBHOOK_BASE_URL=${webhookBaseURL} npx next dev -p 3001`,
    url: baseURL,
    reuseExistingServer: false,
    timeout: 120_000,
  },
})
