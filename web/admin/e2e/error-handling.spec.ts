import { test, expect, type Page } from '@playwright/test'
import { setupAuth, mockAPIError, mockSlowAPI, mockEmptyEndpoints } from './helpers'

/**
 * Cross-cutting error-handling spec.
 *
 * Documented behavior (verified in src/lib/api.ts + page components):
 * - List query failures (500/403) have NO dedicated error UI or toast: React
 *   Query retries once, `data` stays undefined and pages fall back to their
 *   EMPTY state. The tests below assert that fallback (known gap).
 * - 401 responses trigger a token refresh; when the refresh also fails the
 *   client clears tokens and does `window.location.href = '/login'`.
 * - Mutations: contacts page mutations have no onError handler (dialog simply
 *   stays open). Users page mutations call toastError('Error', message) with
 *   the backend `error.message` and keep the dialog open.
 * - Loading state: pages render <Skeleton> placeholders (class `animate-pulse`).
 */

const errorBody = (status: number, message: string) =>
  JSON.stringify({
    success: false,
    error: { code: String(status), message },
    message,
  })

/**
 * Fails only the given HTTP method; other methods fall through to previously
 * registered routes. Register AFTER the generic list mocks.
 */
async function mockMethodError(
  page: Page,
  pattern: string,
  method: string,
  status: number,
  message: string
) {
  await page.route(pattern, async (route) => {
    if (route.request().method() !== method) {
      await route.fallback()
      return
    }
    await route.fulfill({
      status,
      contentType: 'application/json',
      body: errorBody(status, message),
    })
  })
}

async function mockUsersList(page: Page) {
  await page.route('**/api/v1/users**', async (route) => {
    if (route.request().method() !== 'GET') {
      await route.fallback()
      return
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: [
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
        meta: { page: 1, page_size: 20, total_items: 1 },
      }),
    })
  })
}

test.describe('Error Handling - Contacts', () => {
  test('contacts list 500 falls back to the empty state', async ({ page }) => {
    await setupAuth(page)
    await mockEmptyEndpoints(page)
    // Registered last so it wins over the generic contacts mock.
    await mockAPIError(page, 'contacts**', 500)

    await page.goto('/contacts')

    // Known gap: list errors render the empty state, not an error message.
    await expect(page.getByText('No contacts found')).toBeVisible({ timeout: 15000 })
  })

  test('contacts list shows skeleton loading state while the API is slow', async ({ page }) => {
    await setupAuth(page)
    await mockEmptyEndpoints(page)
    await mockSlowAPI(page, 'contacts**', 10_000)

    await page.goto('/contacts')

    await expect(page.locator('.animate-pulse').first()).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('No contacts found')).not.toBeVisible()
  })

  test('401 with failing refresh redirects to login', async ({ page }) => {
    await setupAuth(page)
    await mockEmptyEndpoints(page)
    await mockAPIError(page, 'contacts**', 401, 'token expired')
    // Override the refresh mock from setupAuth (last registered route wins).
    await page.route('**/api/v1/auth/refresh', async (route) => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: errorBody(401, 'invalid refresh token'),
      })
    })

    await page.goto('/contacts')

    // api.ts clears tokens and does window.location.href = '/login'. The
    // full-page navigation can abort in-flight requests, so poll the URL
    // instead of waitForURL (which fails on ERR_ABORTED).
    await expect.poll(() => page.url(), { timeout: 15000 }).toContain('/login')
  })

  test('failed contact creation keeps the dialog open with the typed data', async ({ page }) => {
    await setupAuth(page)
    await mockEmptyEndpoints(page)
    await mockMethodError(page, '**/api/v1/contacts', 'POST', 500, 'Internal server error')

    await page.goto('/contacts')

    await page.getByRole('button', { name: 'Add Contact' }).first().click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 15000 })

    await page.locator('#contact-name').fill('Maria Silva')

    const failedPost = page.waitForResponse(
      (response) =>
        response.url().includes('/api/v1/contacts') &&
        response.request().method() === 'POST' &&
        response.status() === 500
    )
    await page.getByRole('dialog').locator('button[type="submit"]').click()
    await failedPost

    // Known gap: this mutation has no onError handler, so no toast is shown.
    // The dialog only closes onSuccess, so the form must still be there.
    await expect(page.getByRole('dialog')).toBeVisible()
    await expect(page.locator('#contact-name')).toHaveValue('Maria Silva')
  })
})

test.describe('Error Handling - Conversations', () => {
  test('conversations list 500 falls back to the empty state', async ({ page }) => {
    await setupAuth(page)
    await mockEmptyEndpoints(page)
    await mockAPIError(page, 'conversations**', 500)

    await page.goto('/conversations')

    await expect(page.getByText('No conversations found')).toBeVisible({ timeout: 15000 })
  })

  test('conversations list shows skeleton loading state while the API is slow', async ({ page }) => {
    await setupAuth(page)
    await mockEmptyEndpoints(page)
    await mockSlowAPI(page, 'conversations**', 10_000)

    await page.goto('/conversations')

    await expect(page.locator('.animate-pulse').first()).toBeVisible({ timeout: 15000 })
  })
})

test.describe('Error Handling - Users (Team)', () => {
  test('users list 403 falls back to the empty state', async ({ page }) => {
    await setupAuth(page)
    await mockEmptyEndpoints(page)
    await mockAPIError(page, 'users**', 403, 'Access denied')

    await page.goto('/users')

    // Known gap: a forbidden list renders the same empty state as "no data".
    await expect(page.getByText('No team members')).toBeVisible({ timeout: 15000 })
  })

  test('failed user creation shows an error toast and keeps the dialog open', async ({ page }) => {
    await setupAuth(page)
    await mockEmptyEndpoints(page)
    await mockUsersList(page)
    await mockMethodError(page, '**/api/v1/users', 'POST', 500, 'Internal server error')

    await page.goto('/users')

    await page.getByRole('button', { name: 'Add Member' }).first().click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 15000 })

    await page.locator('#name').fill('New Agent')
    await page.locator('#email').fill('new.agent@demo.com')
    // Fills password + confirmation with a strong value, enabling submit.
    await page.getByRole('button', { name: 'Generate Password' }).click()
    await page.getByRole('button', { name: 'Add User' }).click()

    // toastError('Error', error.message) — message comes from the API body.
    await expect(page.getByText('Internal server error').first()).toBeVisible({ timeout: 15000 })
    await expect(page.getByRole('dialog')).toBeVisible()
    await expect(page.locator('#name')).toHaveValue('New Agent')
  })

  test('failed user deletion shows an error toast and keeps the user listed', async ({ page }) => {
    await setupAuth(page)
    await mockEmptyEndpoints(page)
    await mockUsersList(page)
    await mockMethodError(page, '**/api/v1/users/**', 'DELETE', 500, 'Internal server error')

    await page.goto('/users')

    await expect(page.getByText('Agent Smith')).toBeVisible({ timeout: 15000 })

    // The only button inside the user card is the actions dropdown trigger.
    const card = page
      .locator('[class*="hover:border-primary"]')
      .filter({ hasText: 'Agent Smith' })
    await card.locator('button[aria-haspopup="menu"]').click()
    await page.getByRole('menuitem', { name: 'Delete' }).click()
    await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click()

    await expect(page.getByText('Internal server error').first()).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('Agent Smith')).toBeVisible()
  })
})
