import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

async function mockAnalyticsEndpoints(page: import('@playwright/test').Page) {
  // Mock analytics overview with stats data
  await page.route('**/api/v1/analytics/overview**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          total_conversations: 150,
          total_messages: 1200,
          avg_response_time: 120,
          avg_first_response_ms: 120000,
          resolution_rate: 85.5,
          conversations_trend: 12.3,
          resolution_trend: 5.2,
          avg_confidence: 0.92,
        },
      }),
    })
  })

  // Mock analytics conversations
  await page.route('**/api/v1/analytics/conversations**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: { data: [] } }),
    })
  })

  // Mock analytics flows
  await page.route('**/api/v1/analytics/flows**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: { data: [] } }),
    })
  })

  // Mock analytics escalations
  await page.route('**/api/v1/analytics/escalations**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: { data: [] } }),
    })
  })

  // Mock analytics channels
  await page.route('**/api/v1/analytics/channels**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: { data: [] } }),
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

test.describe('Analytics Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockAnalyticsEndpoints(page)
  })

  test('renders analytics page', async ({ page }) => {
    await page.goto('/analytics')

    await expect(page.getByText('Analytics').first()).toBeVisible({ timeout: 15000 })
  })

  test('shows stats cards', async ({ page }) => {
    await page.goto('/analytics')

    await expect(page.getByText('Analytics').first()).toBeVisible({ timeout: 15000 })

    // Stats cards should show overview values
    await expect(page.getByText('150').first()).toBeVisible()
    await expect(page.getByText('85.5%').first()).toBeVisible()
    await expect(page.getByText('Total Conversations').first()).toBeVisible()
    await expect(page.getByText('Resolution Rate').first()).toBeVisible()
  })

  test('shows date range picker', async ({ page }) => {
    await page.goto('/analytics')

    await expect(page.getByText('Analytics').first()).toBeVisible({ timeout: 15000 })

    // Date range picker should be visible with date inputs
    await expect(page.locator('input[type="date"]').first()).toBeVisible()
  })

  test('shows period options', async ({ page }) => {
    await page.goto('/analytics')

    await expect(page.getByText('Analytics').first()).toBeVisible({ timeout: 15000 })

    // Period selector should contain the period options
    const periodSelect = page.locator('select').first()
    await expect(periodSelect).toBeVisible()

    // Check that the period options exist
    await expect(page.locator('option[value="daily"]')).toBeAttached()
    await expect(page.locator('option[value="weekly"]')).toBeAttached()
    await expect(page.locator('option[value="monthly"]')).toBeAttached()
  })
})
