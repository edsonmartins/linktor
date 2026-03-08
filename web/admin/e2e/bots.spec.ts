import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

async function mockBots(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/bots**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          data: [
            {
              id: 'bot-1',
              tenant_id: 'tenant-1',
              name: 'Atendimento ao Cliente',
              type: 'conversational',
              provider: 'openai',
              model: 'gpt-4',
              status: 'active',
              channels: ['ch-1'],
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-15T00:00:00Z',
            },
            {
              id: 'bot-2',
              tenant_id: 'tenant-1',
              name: 'Bot de Vendas',
              type: 'conversational',
              provider: 'anthropic',
              model: 'claude-3-sonnet',
              status: 'inactive',
              channels: [],
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

  for (const endpoint of ['conversations', 'channels', 'contacts', 'flows']) {
    await page.route(`**/api/v1/${endpoint}**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      })
    })
  }
}

test.describe('Bots Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockBots(page)
  })

  test('lists bots', async ({ page }) => {
    await page.goto('/bots')

    await expect(page.getByText('Atendimento ao Cliente')).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('Bot de Vendas')).toBeVisible()
  })

  test('shows bot status', async ({ page }) => {
    await page.goto('/bots')

    await expect(page.getByText('Atendimento ao Cliente')).toBeVisible({ timeout: 15000 })
    await expect(page.getByText(/active/i).first()).toBeVisible()
  })

  test('shows bot provider', async ({ page }) => {
    await page.goto('/bots')

    await expect(page.getByText('Atendimento ao Cliente')).toBeVisible({ timeout: 15000 })
    await expect(page.getByText(/openai|gpt-4/i).first()).toBeVisible()
  })
})
