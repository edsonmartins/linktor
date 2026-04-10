import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

async function mockContacts(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/contacts**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: [
          {
            id: 'contact-1',
            tenant_id: 'tenant-1',
            name: 'Carlos Oliveira',
            email: 'carlos@example.com',
            phone: '+5511999999999',
            tags: ['vip', 'recorrente'],
            identities: [],
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-15T00:00:00Z',
          },
          {
            id: 'contact-2',
            tenant_id: 'tenant-1',
            name: 'Ana Pereira',
            email: 'ana@example.com',
            phone: '+5511888888888',
            tags: [],
            identities: [],
            created_at: '2024-01-05T00:00:00Z',
            updated_at: '2024-01-10T00:00:00Z',
          },
        ],
        meta: {
          page: 1,
          page_size: 20,
          total_pages: 1,
          total_items: 2,
          has_next: false,
          has_previous: false,
        },
      }),
    })
  })

  for (const endpoint of ['conversations', 'channels', 'bots', 'flows']) {
    await page.route(`**/api/v1/${endpoint}**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      })
    })
  }
}

test.describe('Contacts Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockContacts(page)
  })

  test('lists contacts', async ({ page }) => {
    await page.goto('/contacts')

    await expect(page.getByText('Carlos Oliveira')).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('Ana Pereira')).toBeVisible()
  })

  test('shows contact phone', async ({ page }) => {
    await page.goto('/contacts')

    await expect(page.getByText('Carlos Oliveira')).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('+5511999999999')).toBeVisible()
  })

  test('shows contact email', async ({ page }) => {
    await page.goto('/contacts')

    await expect(page.getByText('Carlos Oliveira')).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('carlos@example.com')).toBeVisible()
  })
})
