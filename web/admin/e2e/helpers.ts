import type { Page } from '@playwright/test'

const mockUser = {
  id: 'user-1',
  tenant_id: 'tenant-1',
  email: 'admin@demo.com',
  name: 'Admin User',
  role: 'admin',
  avatar_url: null,
}

/**
 * Sets up authenticated state in localStorage and mocks common API routes.
 * Must be called BEFORE page.goto().
 */
export async function setupAuth(page: Page) {
  // Set tokens + zustand persisted auth store
  await page.addInitScript(() => {
    localStorage.setItem('access_token', 'mock-access-token')
    localStorage.setItem('refresh_token', 'mock-refresh-token')
    localStorage.setItem(
      'auth-storage',
      JSON.stringify({
        state: {
          user: {
            id: 'user-1',
            tenant_id: 'tenant-1',
            email: 'admin@demo.com',
            name: 'Admin User',
            role: 'admin',
          },
          isAuthenticated: true,
        },
        version: 0,
      })
    )
  })

  // Mock auth refresh
  await page.route('**/api/v1/auth/refresh', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          access_token: 'mock-access-token-refreshed',
          refresh_token: 'mock-refresh-token-refreshed',
          user: mockUser,
        },
      }),
    })
  })

  // Mock /me
  await page.route('**/api/v1/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: mockUser }),
    })
  })

  // Mock stats
  await page.route('**/api/v1/stats**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: {} }),
    })
  })
}

/**
 * Mock empty list endpoints to prevent 404s.
 */
export async function mockEmptyEndpoints(page: Page) {
  for (const endpoint of ['conversations', 'channels', 'contacts', 'bots', 'flows', 'knowledge-bases']) {
    await page.route(`**/api/v1/${endpoint}**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: [],
          meta: { page: 1, page_size: 20, total_items: 0 },
        }),
      })
    })
  }
}
