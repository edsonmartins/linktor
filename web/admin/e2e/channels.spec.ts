import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

async function mockChannels(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/channels**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: [
          {
            id: 'ch-1',
            type: 'whatsapp_official',
            name: 'WhatsApp Business',
            identifier: '+5511999999999',
            enabled: true,
            connection_status: 'connected',
            created_at: '2024-01-01T00:00:00Z',
          },
          {
            id: 'ch-2',
            type: 'telegram',
            name: 'Telegram Bot',
            identifier: '@mybot',
            enabled: false,
            connection_status: 'disconnected',
            created_at: '2024-01-02T00:00:00Z',
          },
        ],
        meta: { page: 1, page_size: 20, total_items: 2 },
      }),
    })
  })

  // Mock other endpoints
  for (const endpoint of ['conversations', 'contacts', 'bots', 'flows']) {
    await page.route(`**/api/v1/${endpoint}**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [], meta: { page: 1, page_size: 20, total_items: 0 } }),
      })
    })
  }
}

test.describe('Channels Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockChannels(page)
  })

  test('lists existing channels', async ({ page }) => {
    await page.goto('/channels')

    await expect(page.locator('h3').filter({ hasText: 'WhatsApp Business' })).toBeVisible({ timeout: 15000 })
    await expect(page.locator('h3').filter({ hasText: 'Telegram Bot' })).toBeVisible()
  })

  test('shows channel type badges', async ({ page }) => {
    await page.goto('/channels')

    await expect(page.getByText(/whatsapp/i).first()).toBeVisible({ timeout: 15000 })
  })

  test('shows channel status', async ({ page }) => {
    await page.goto('/channels')

    await expect(page.locator('h3').filter({ hasText: 'WhatsApp Business' })).toBeVisible({ timeout: 15000 })
    await expect(page.getByText(/connected/i).first()).toBeVisible()
  })
})
