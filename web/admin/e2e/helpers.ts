import type { Page } from '@playwright/test'

export type UserRole = 'admin' | 'supervisor' | 'agent' | 'viewer'

const baseMockUser = {
  id: 'user-1',
  tenant_id: 'tenant-1',
  email: 'admin@demo.com',
  name: 'Admin User',
  avatar_url: null,
}

/**
 * Sets up authenticated state in localStorage and mocks common API routes.
 * Must be called BEFORE page.goto().
 */
export async function setupAuth(page: Page, role: UserRole = 'admin') {
  const mockUser = { ...baseMockUser, role }

  // Auth marker cookie must exist before navigation: middleware.ts redirects
  // to /login server-side when it is missing.
  await page.context().addCookies([
    { name: 'linktor_authed', value: '1', url: 'http://localhost:3000' },
  ])

  // Set tokens + zustand persisted auth store
  await page.addInitScript((user) => {
    localStorage.setItem('access_token', 'mock-access-token')
    localStorage.setItem('refresh_token', 'mock-refresh-token')
    localStorage.setItem(
      'auth-storage',
      JSON.stringify({
        state: {
          user,
          isAuthenticated: true,
        },
        version: 0,
      })
    )
  }, mockUser)

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

  // Mock logout: the real backend returns 200 and expires the auth cookies. Without
  // this, the POST /auth/logout is unmocked → the api client treats it as 401 →
  // triggers the (mocked) refresh → re-authenticates mid-logout and lands on
  // /dashboard instead of /login. Return 200 and clear the JS-readable marker.
  await page.route('**/api/v1/auth/logout', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'Set-Cookie': 'linktor_authed=; Path=/; Max-Age=0' },
      contentType: 'application/json',
      body: JSON.stringify({ success: true }),
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
 * Convenience wrappers for non-admin roles.
 */
export const setupAgentAuth = (page: Page) => setupAuth(page, 'agent')
export const setupViewerAuth = (page: Page) => setupAuth(page, 'viewer')

/**
 * Makes an endpoint fail with the given HTTP status. Register AFTER other
 * routes for the same pattern so it takes precedence (Playwright matches the
 * most recently registered route first).
 */
export async function mockAPIError(
  page: Page,
  endpoint: string,
  status: number,
  message = 'Something went wrong'
) {
  await page.route(`**/api/v1/${endpoint}`, async (route) => {
    await route.fulfill({
      status,
      contentType: 'application/json',
      body: JSON.stringify({
        success: false,
        error: { code: String(status), message },
        message,
      }),
    })
  })
}

/**
 * Mocks an endpoint that never responds within the test's patience, to
 * exercise loading states. Aborts after `delayMs` to avoid leaking requests.
 */
export async function mockSlowAPI(page: Page, endpoint: string, delayMs = 5_000) {
  await page.route(`**/api/v1/${endpoint}`, async (route) => {
    await new Promise((resolve) => setTimeout(resolve, delayMs))
    await route.abort('timedout')
  })
}

/**
 * Fulfills a list endpoint with the provided items in the standard envelope.
 */
export async function mockList(
  page: Page,
  endpoint: string,
  items: unknown[],
  meta: Record<string, unknown> = {}
) {
  await page.route(`**/api/v1/${endpoint}`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: items,
        meta: { page: 1, page_size: 20, total_items: items.length, ...meta },
      }),
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
