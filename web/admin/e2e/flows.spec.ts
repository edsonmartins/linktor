import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

async function mockFlows(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/flows**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          data: [
            {
              id: 'flow-1',
              name: 'Welcome Flow',
              trigger: 'message_received',
              trigger_value: 'hello',
              is_active: true,
              nodes: [{}, {}],
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-15T00:00:00Z',
            },
            {
              id: 'flow-2',
              name: 'Order Status',
              trigger: 'keyword',
              trigger_value: 'order',
              is_active: false,
              nodes: [{}],
              created_at: '2024-01-05T00:00:00Z',
              updated_at: '2024-01-10T00:00:00Z',
            },
          ],
          total: 2,
          page: 1,
          per_page: 20,
          total_pages: 1,
        },
      }),
    })
  })

  for (const endpoint of ['conversations', 'channels', 'contacts', 'bots']) {
    await page.route(`**/api/v1/${endpoint}**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      })
    })
  }
}

test.describe('Flows Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockFlows(page)
  })

  test('lists flows', async ({ page }) => {
    await page.goto('/flows')

    await expect(page.getByText('Welcome Flow')).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('Order Status')).toBeVisible()
  })

  test('shows flow status', async ({ page }) => {
    await page.goto('/flows')

    await expect(page.getByText('Welcome Flow')).toBeVisible({ timeout: 15000 })
    await expect(page.getByText(/active/i).first()).toBeVisible()
    await expect(page.getByText(/inactive/i).first()).toBeVisible()
  })

  test('shows search input', async ({ page }) => {
    await page.goto('/flows')

    await expect(page.getByText('Welcome Flow')).toBeVisible({ timeout: 15000 })
    await expect(page.getByPlaceholder(/search flows/i)).toBeVisible()
  })

  test('has create flow button', async ({ page }) => {
    await page.goto('/flows')

    await expect(page.getByText('Welcome Flow')).toBeVisible({ timeout: 15000 })
    await expect(
      page.getByRole('button', { name: /add flow|new flow|create flow|\+/i })
    ).toBeVisible()
  })
})
