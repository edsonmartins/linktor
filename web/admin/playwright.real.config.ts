import { defineConfig, devices } from '@playwright/test'

const port = process.env.PLAYWRIGHT_PORT || '3001'
const baseURL = process.env.PLAYWRIGHT_BASE_URL || `http://localhost:${port}`
const apiURL = (process.env.E2E_REAL_API_BASE || process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081/api/v1').replace(/\/$/, '')
const inferredBaseURL = apiURL.replace(/^http/, 'ws').replace(/\/api\/v1\/?$/, '/api/v1/ws')
const inferredWebhookBaseURL = apiURL.replace(/\/api\/v1\/?$/, '')
const wsURL = process.env.NEXT_PUBLIC_WS_URL || inferredBaseURL
const webhookBaseURL = process.env.NEXT_PUBLIC_WEBHOOK_BASE_URL || inferredWebhookBaseURL

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
    command: `NEXT_PUBLIC_API_URL=${apiURL} NEXT_PUBLIC_WS_URL=${wsURL} NEXT_PUBLIC_WEBHOOK_BASE_URL=${webhookBaseURL} npx next dev -p ${port}`,
    url: baseURL,
    reuseExistingServer: false,
    timeout: 120_000,
  },
})
