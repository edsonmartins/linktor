import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

async function mockKnowledgeBases(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/knowledge-bases**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          data: [
            {
              id: 'kb-1',
              name: 'Product FAQ',
              type: 'faq',
              status: 'active',
              item_count: 25,
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-15T00:00:00Z',
            },
            {
              id: 'kb-2',
              name: 'Technical Docs',
              type: 'document',
              status: 'processing',
              item_count: 12,
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

  for (const endpoint of ['conversations', 'channels', 'contacts', 'bots', 'flows']) {
    await page.route(`**/api/v1/${endpoint}**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      })
    })
  }
}

test.describe('Knowledge Base Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockKnowledgeBases(page)
  })

  test('lists knowledge bases', async ({ page }) => {
    await page.goto('/knowledge-base')

    await expect(page.getByText('Product FAQ')).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('Technical Docs')).toBeVisible()
  })

  test('shows knowledge base status', async ({ page }) => {
    await page.goto('/knowledge-base')

    await expect(page.getByText('Product FAQ')).toBeVisible({ timeout: 15000 })
    await expect(page.getByText(/active/i).first()).toBeVisible()
  })

  test('has create button', async ({ page }) => {
    await page.goto('/knowledge-base')

    await expect(page.getByText('Product FAQ')).toBeVisible({ timeout: 15000 })
    await expect(
      page.getByRole('button', { name: /add knowledge|new knowledge base|create knowledge base|\+/i })
    ).toBeVisible()
  })

  test('shows search input', async ({ page }) => {
    await page.goto('/knowledge-base')

    await expect(page.getByText('Product FAQ')).toBeVisible({ timeout: 15000 })
    await expect(page.getByPlaceholder(/search knowledge bases/i)).toBeVisible()
  })
})
