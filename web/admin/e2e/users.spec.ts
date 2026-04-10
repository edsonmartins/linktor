import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

async function mockUsers(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/users**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: [
          {
            id: 'user-1',
            tenant_id: 'tenant-1',
            name: 'Admin User',
            email: 'admin@demo.com',
            role: 'admin',
            status: 'active',
            avatar_url: null,
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
          },
          {
            id: 'user-2',
            tenant_id: 'tenant-1',
            name: 'Agent Smith',
            email: 'agent@demo.com',
            role: 'agent',
            status: 'active',
            avatar_url: null,
            created_at: '2024-01-05T00:00:00Z',
            updated_at: '2024-01-05T00:00:00Z',
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

test.describe('Users Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockUsers(page)
  })

  test('lists users', async ({ page }) => {
    await page.goto('/users')

    await expect(page.getByRole('heading', { name: 'Admin User' })).toBeVisible({ timeout: 15000 })
    await expect(page.getByRole('heading', { name: 'Agent Smith' })).toBeVisible()
  })

  test('shows user roles', async ({ page }) => {
    await page.goto('/users')

    await expect(page.getByRole('heading', { name: 'Admin User' })).toBeVisible({ timeout: 15000 })
    await expect(page.getByText(/admin/i).first()).toBeVisible()
    await expect(page.getByText(/agent/i).first()).toBeVisible()
  })

  test('shows user emails', async ({ page }) => {
    await page.goto('/users')

    await expect(page.getByRole('heading', { name: 'Admin User' })).toBeVisible({ timeout: 15000 })
    await expect(page.getByRole('main').getByText('admin@demo.com')).toBeVisible()
    await expect(page.getByRole('main').getByText('agent@demo.com')).toBeVisible()
  })

  test('has invite/add user button', async ({ page }) => {
    await page.goto('/users')

    await expect(page.getByRole('heading', { name: 'Admin User' })).toBeVisible({ timeout: 15000 })
    await expect(
      page.getByRole('button', { name: /add member|add user|invite/i })
    ).toBeVisible()
  })
})
