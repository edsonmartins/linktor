import { test, expect, type Page } from '@playwright/test'
import { setupAuth, setupAgentAuth, setupViewerAuth, mockEmptyEndpoints } from './helpers'

/**
 * RBAC behavior spec.
 *
 * Frontend role gating mirrors the backend (RequireRole("admin","owner")):
 * - The sidebar hides admin-only entries (Team, Roles, Audit Log,
 *   Observability) from non-admin roles — see src/components/layout/sidebar.tsx
 *   and src/lib/rbac.ts.
 * - RoleGuard (src/components/role-guard.tsx) blocks direct-URL access to those
 *   routes with an access-denied screen.
 * - Forbidden actions on allowed pages still surface the backend 403 as an
 *   error toast (global react-query onError + per-mutation handlers).
 * - The Settings profile section shows the current role in a Badge.
 */

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

test.describe('RBAC', () => {
  test('agent sees their role badge on the settings page', async ({ page }) => {
    await setupAgentAuth(page)
    await mockEmptyEndpoints(page)

    await page.goto('/settings')

    await expect(page.getByText('Role', { exact: true })).toBeVisible({ timeout: 15000 })
    // Badge renders the raw role value (visually capitalized via CSS only).
    await expect(page.getByText('agent', { exact: true })).toBeVisible()
  })

  test('viewer sees their role badge on the settings page', async ({ page }) => {
    await setupViewerAuth(page)
    await mockEmptyEndpoints(page)

    await page.goto('/settings')

    await expect(page.getByText('Role', { exact: true })).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('viewer', { exact: true })).toBeVisible()
  })

  test('sidebar hides admin-only navigation from an agent', async ({ page }) => {
    await setupAgentAuth(page)
    await mockEmptyEndpoints(page)

    // /settings renders the same sidebar and is quieter than /dashboard
    // (no widget polling re-renders detaching elements mid-test).
    await page.goto('/settings')

    // Non-admin tools the agent CAN use stay visible.
    await expect(page.getByRole('link', { name: 'Contacts' })).toBeVisible({ timeout: 15000 })
    await expect(page.getByRole('link', { name: 'Conversations' })).toBeVisible()
    await expect(page.getByRole('link', { name: 'Settings' })).toBeVisible()

    // Admin-only entries are hidden (the API enforces this too).
    await expect(page.getByRole('link', { name: 'Team' })).toHaveCount(0)
    await expect(page.getByRole('link', { name: 'Roles' })).toHaveCount(0)
    await expect(page.getByRole('link', { name: 'Audit Log' })).toHaveCount(0)
    await expect(page.getByRole('link', { name: 'Observability' })).toHaveCount(0)
  })

  test('admin sees the admin-only navigation', async ({ page }) => {
    await setupAuth(page) // default role: admin
    await mockEmptyEndpoints(page)

    await page.goto('/settings')

    await expect(page.getByRole('link', { name: 'Team' })).toBeVisible({ timeout: 15000 })
    await expect(page.getByRole('link', { name: 'Roles' })).toBeVisible()
    await expect(page.getByRole('link', { name: 'Audit Log' })).toBeVisible()
  })

  test('agent reaching an admin-only route by URL sees access denied', async ({ page }) => {
    await setupAgentAuth(page)
    await mockEmptyEndpoints(page)
    await mockUsersList(page)

    await page.goto('/users')

    // RoleGuard renders the access-denied screen instead of the page.
    await expect(page.getByText('Access denied', { exact: true })).toBeVisible({ timeout: 15000 })
    // The team list never renders for a non-admin.
    await expect(page.getByText('Agent Smith')).toHaveCount(0)
  })

  test('logout from the sidebar returns to the login page', async ({ page }) => {
    await setupAgentAuth(page)
    await mockEmptyEndpoints(page)

    // /settings renders the same sidebar and is quieter than /dashboard.
    await page.goto('/settings')

    await expect(page.getByRole('button', { name: 'Logout' })).toBeVisible({ timeout: 15000 })

    // After logout the refresh-token cookie is gone, so the AuthGuard's on-mount
    // re-auth refresh must fail. Without this override the refresh mock still
    // returns 200, re-authenticating the just-logged-out user and bouncing them
    // to /dashboard. Re-route refresh to 401 to mirror the cleared session.
    await page.route('**/api/v1/auth/refresh', async (route) => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ success: false, error: { code: '401', message: 'session ended' } }),
      })
    })

    await page.getByRole('button', { name: 'Logout' }).click()

    // AuthGuard redirects unauthenticated users to /login (with returnUrl).
    await expect.poll(() => page.url(), { timeout: 15000 }).toContain('/login')
  })
})
