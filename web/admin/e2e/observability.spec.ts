import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

async function mockObservabilityEndpoints(page: import('@playwright/test').Page) {
  // Mock specific observability endpoints first (Playwright matches routes in registration order)
  await page.route('**/api/v1/observability/logs**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: { data: [], total: 0, page: 1, per_page: 50 },
      }),
    })
  })

  await page.route('**/api/v1/observability/queue**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: { streams: [], consumers: [] },
      }),
    })
  })

  await page.route('**/api/v1/observability/stats**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: {} }),
    })
  })

  // Catch-all for any other observability endpoints
  await page.route('**/api/v1/observability/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: {} }),
    })
  })

  // Mock other common endpoints to prevent 404s
  for (const endpoint of ['conversations', 'channels', 'contacts', 'bots', 'flows', 'knowledge-bases']) {
    await page.route(`**/api/v1/${endpoint}**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      })
    })
  }
}

test.describe('Observability Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockObservabilityEndpoints(page)
  })

  test('renders observability page', async ({ page }) => {
    await page.goto('/observability')

    await expect(page.getByText('Observability').first()).toBeVisible({ timeout: 15000 })
  })

  test('shows tabs', async ({ page }) => {
    await page.goto('/observability')

    await expect(page.getByText('Observability').first()).toBeVisible({ timeout: 15000 })

    // Should show all three tabs
    await expect(page.getByText('Logs').first()).toBeVisible()
    await expect(page.getByText('Message Queues').first()).toBeVisible()
    await expect(page.getByText('Statistics').first()).toBeVisible()
  })

  test('can switch tabs', async ({ page }) => {
    await page.goto('/observability')

    await expect(page.getByText('Observability').first()).toBeVisible({ timeout: 15000 })

    // Logs tab should be active by default (tab content visible)
    const logsTab = page.getByRole('tab', { name: /Logs/i })
    await expect(logsTab).toBeVisible()

    // Click on Message Queues tab
    const queueTab = page.getByRole('tab', { name: /Message Queues/i })
    await queueTab.click()

    // The queue tab panel content should now be visible
    await expect(page.getByRole('tabpanel', { name: /Message Queues/i })).toBeVisible()

    // Click on Statistics tab
    const statsTab = page.getByRole('tab', { name: /Statistics/i })
    await statsTab.click()

    // The stats tab panel content should now be visible
    await expect(page.getByRole('tabpanel', { name: /Statistics/i })).toBeVisible()
  })
})
