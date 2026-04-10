import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

async function mockDashboardEndpoints(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/analytics/overview**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          total_conversations: 12,
          resolution_rate: 0.85,
          avg_first_response_ms: 42000,
        },
      }),
    })
  })

  await page.route('**/api/v1/conversations**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: [],
        meta: {
          page: 1,
          page_size: 20,
          total_pages: 0,
          total_items: 0,
          has_next: false,
          has_previous: false,
        },
      }),
    })
  })

  await page.route('**/api/v1/channels**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: [],
      }),
    })
  })

  await page.route('**/api/v1/bots**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          data: [],
          pagination: {
            total: 0,
            page: 1,
            page_size: 20,
            pages: 0,
          },
        },
      }),
    })
  })

  await page.route('**/api/v1/contacts**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: [],
        meta: {
          page: 1,
          page_size: 1,
          total_pages: 0,
          total_items: 0,
          has_next: false,
          has_previous: false,
        },
      }),
    })
  })
}

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockDashboardEndpoints(page)
  })

  test('loads dashboard page', async ({ page }) => {
    await page.goto('/dashboard')
    await expect(page).toHaveURL(/dashboard/, { timeout: 15000 })
  })

  test('sidebar navigation is visible', async ({ page }) => {
    await page.goto('/dashboard')

    // Sidebar links use href-based navigation; pt-BR labels
    // Check that a link to /conversations exists
    await expect(page.locator('a[href="/conversations"]')).toBeVisible({ timeout: 15000 })
  })

  test('can navigate to channels page', async ({ page }) => {
    await page.goto('/dashboard')

    await page.locator('a[href="/channels"]').click({ timeout: 15000 })
    await expect(page).toHaveURL(/channels/, { timeout: 10000 })
  })

  test('can navigate to contacts page', async ({ page }) => {
    await page.goto('/dashboard')

    await page.locator('a[href="/contacts"]').click({ timeout: 15000 })
    await expect(page).toHaveURL(/contacts/, { timeout: 10000 })
  })
})
