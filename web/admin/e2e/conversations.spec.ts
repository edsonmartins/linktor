import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

async function mockConversations(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/conversations**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: [
          {
            id: 'conv-1',
            channel_id: 'ch-1',
            contact_id: 'contact-1',
            status: 'open',
            priority: 'normal',
            unread_count: 3,
            created_at: '2024-01-15T10:00:00Z',
            updated_at: '2024-01-15T12:00:00Z',
            last_message_at: '2024-01-15T12:00:00Z',
            contact: {
              id: 'contact-1',
              name: 'João Silva',
              phone: '+5511999999999',
            },
            channel: {
              id: 'ch-1',
              type: 'whatsapp_official',
            },
            last_message: {
              content: 'Olá, preciso de ajuda',
              created_at: '2024-01-15T12:00:00Z',
            },
          },
          {
            id: 'conv-2',
            channel_id: 'ch-1',
            contact_id: 'contact-2',
            status: 'resolved',
            priority: 'high',
            unread_count: 0,
            created_at: '2024-01-14T08:00:00Z',
            updated_at: '2024-01-14T18:00:00Z',
            last_message_at: '2024-01-14T18:00:00Z',
            contact: {
              id: 'contact-2',
              name: 'Maria Santos',
              phone: '+5511888888888',
            },
            channel: {
              id: 'ch-1',
              type: 'whatsapp_official',
            },
            last_message: {
              content: 'Obrigada pela ajuda!',
              created_at: '2024-01-14T18:00:00Z',
            },
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

  // Mock other endpoints
  for (const endpoint of ['channels', 'contacts', 'bots', 'flows']) {
    await page.route(`**/api/v1/${endpoint}**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      })
    })
  }
}

test.describe('Conversations Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockConversations(page)
  })

  test('lists conversations', async ({ page }) => {
    await page.goto('/conversations')

    await expect(page.getByText('João Silva')).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('Maria Santos')).toBeVisible()
  })

  test('shows last message preview', async ({ page }) => {
    await page.goto('/conversations')

    await expect(page.getByText(/preciso de ajuda/)).toBeVisible({ timeout: 15000 })
  })

  test('shows unread count badge', async ({ page }) => {
    await page.goto('/conversations')

    // Wait for conversations to load
    await expect(page.getByText('João Silva')).toBeVisible({ timeout: 15000 })

    // Should show unread indicator for conv-1 (unread_count: 3)
    await expect(page.getByText('3').first()).toBeVisible()
  })
})
