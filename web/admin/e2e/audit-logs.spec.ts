import { expect, test } from '@playwright/test'
import { mockEmptyEndpoints, setupAuth } from './helpers'

const AUDIT_LOGS = [
  {
    id: 'log-1',
    tenant_id: 'tenant-1',
    actor_id: 'user-1',
    actor_email: 'admin@demo.com',
    actor_name: 'Admin User',
    action: 'create',
    resource_type: 'contact',
    resource_id: 'abcdef12-3456-7890-aaaa-bbbbccccdddd',
    ip_address: '10.0.0.1',
    created_at: '2026-06-01T10:00:00Z',
  },
  {
    id: 'log-2',
    tenant_id: 'tenant-1',
    actor_id: 'user-2',
    actor_email: 'agent@demo.com',
    actor_name: 'Agent Smith',
    action: 'update',
    resource_type: 'conversation',
    resource_id: 'feed1234-0000-0000-0000-000000000000',
    ip_address: '10.0.0.2',
    created_at: '2026-06-02T11:30:00Z',
  },
  {
    id: 'log-3',
    tenant_id: 'tenant-1',
    actor_id: 'user-1',
    actor_email: 'admin@demo.com',
    actor_name: 'Admin User',
    action: 'delete',
    resource_type: 'role',
    resource_id: 'cafe5678-0000-0000-0000-000000000000',
    ip_address: '10.0.0.3',
    created_at: '2026-06-03T09:15:00Z',
  },
]

async function mockAuditEndpoints(page: import('@playwright/test').Page) {
  await mockEmptyEndpoints(page)

  await page.route('**/api/v1/audit-logs**', async (route) => {
    const url = new URL(route.request().url())
    let items = [...AUDIT_LOGS]
    const action = url.searchParams.get('action')
    if (action) items = items.filter((l) => l.action.includes(action))
    const resource = url.searchParams.get('resource_type')
    if (resource) items = items.filter((l) => l.resource_type.includes(resource))

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: items,
        meta: {
          page: 1,
          page_size: 50,
          total_pages: 1,
          total_items: items.length,
          has_next: false,
          has_previous: false,
        },
      }),
    })
  })
}

/**
 * Paginated dataset: 120 items / 50 per page = 3 pages. Each page returns a
 * single distinguishable entry so the test can assert the visible page.
 */
async function mockPaginatedAudit(page: import('@playwright/test').Page, urls: string[]) {
  await page.route('**/api/v1/audit-logs**', async (route) => {
    const url = new URL(route.request().url())
    urls.push(url.toString())
    const pageNum = Number(url.searchParams.get('page') ?? '1')

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: [
          {
            id: `log-p${pageNum}`,
            tenant_id: 'tenant-1',
            actor_id: `user-p${pageNum}`,
            actor_email: `page${pageNum}-actor@demo.com`,
            actor_name: `Page ${pageNum} Actor`,
            action: 'login',
            resource_type: 'session',
            resource_id: `00000000-0000-0000-0000-00000000000${pageNum}`,
            ip_address: `10.0.1.${pageNum}`,
            created_at: '2026-06-05T08:00:00Z',
          },
        ],
        meta: {
          page: pageNum,
          page_size: 50,
          total_pages: 3,
          total_items: 120,
          has_next: pageNum < 3,
          has_previous: pageNum > 1,
        },
      }),
    })
  })
}

test.describe('Audit Logs Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockAuditEndpoints(page)
  })

  test('lists audit entries with create/update/delete actions', async ({ page }) => {
    await page.goto('/audit-logs')

    await expect(page.getByRole('heading', { name: 'Audit Log' })).toBeVisible({ timeout: 15000 })

    await expect(page.getByText('create', { exact: true })).toBeVisible()
    await expect(page.getByText('update', { exact: true })).toBeVisible()
    await expect(page.getByText('delete', { exact: true })).toBeVisible()

    await expect(page.getByText('admin@demo.com').first()).toBeVisible()
    await expect(page.getByText('agent@demo.com')).toBeVisible()
  })

  test('shows resource references and IP addresses', async ({ page }) => {
    await page.goto('/audit-logs')

    await expect(page.getByRole('heading', { name: 'Audit Log' })).toBeVisible({ timeout: 15000 })

    // Resource column renders type:first-8-chars-of-id
    await expect(page.getByText('contact:abcdef12')).toBeVisible()
    await expect(page.getByText('conversation:feed1234')).toBeVisible()
    await expect(page.getByText('role:cafe5678')).toBeVisible()

    await expect(page.getByText('10.0.0.1')).toBeVisible()
    await expect(page.getByText('10.0.0.3')).toBeVisible()
  })

  test('filters by action and refetches with the action query param', async ({ page }) => {
    const requestUrls: string[] = []
    await page.route('**/api/v1/audit-logs**', async (route) => {
      requestUrls.push(route.request().url())
      await route.fallback()
    })

    await page.goto('/audit-logs')

    await expect(page.getByText('contact:abcdef12')).toBeVisible({ timeout: 15000 })

    await page.getByPlaceholder('Filter by action').fill('delete')

    await expect
      .poll(() => requestUrls.some((u) => u.includes('action=delete') && u.includes('page=1')))
      .toBe(true)

    await expect(page.getByText('role:cafe5678')).toBeVisible()
    await expect(page.getByText('contact:abcdef12')).toBeHidden()
  })

  test('filters by resource and refetches with the resource_type query param', async ({ page }) => {
    const requestUrls: string[] = []
    await page.route('**/api/v1/audit-logs**', async (route) => {
      requestUrls.push(route.request().url())
      await route.fallback()
    })

    await page.goto('/audit-logs')

    await expect(page.getByText('conversation:feed1234')).toBeVisible({ timeout: 15000 })

    await page.getByPlaceholder('Filter by resource').fill('conversation')

    await expect
      .poll(() => requestUrls.some((u) => u.includes('resource_type=conversation')))
      .toBe(true)

    await expect(page.getByText('conversation:feed1234')).toBeVisible()
    await expect(page.getByText('role:cafe5678')).toBeHidden()
  })

  test('paginates to the next page with page=2 query param', async ({ page }) => {
    const requestUrls: string[] = []
    await mockPaginatedAudit(page, requestUrls)

    await page.goto('/audit-logs')

    await expect(page.getByText('page1-actor@demo.com')).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('120 · 1/3')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Previous' })).toBeDisabled()

    await page.getByRole('button', { name: 'Next', exact: true }).click()

    await expect.poll(() => requestUrls.some((u) => u.includes('page=2'))).toBe(true)

    await expect(page.getByText('page2-actor@demo.com')).toBeVisible()
    await expect(page.getByText('120 · 2/3')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Previous' })).toBeEnabled()
  })

  test('filtering resets pagination back to page 1', async ({ page }) => {
    const requestUrls: string[] = []
    await mockPaginatedAudit(page, requestUrls)

    await page.goto('/audit-logs')

    await expect(page.getByText('page1-actor@demo.com')).toBeVisible({ timeout: 15000 })

    await page.getByRole('button', { name: 'Next', exact: true }).click()
    await expect(page.getByText('page2-actor@demo.com')).toBeVisible()

    await page.getByPlaceholder('Filter by action').fill('login')

    await expect
      .poll(() => requestUrls.some((u) => u.includes('action=login') && u.includes('page=1')))
      .toBe(true)

    await expect(page.getByText('120 · 1/3')).toBeVisible()
  })

  test('shows empty state when there are no entries', async ({ page }) => {
    await page.route('**/api/v1/audit-logs**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: [],
          meta: {
            page: 1,
            page_size: 50,
            total_pages: 0,
            total_items: 0,
            has_next: false,
            has_previous: false,
          },
        }),
      })
    })

    await page.goto('/audit-logs')

    await expect(page.getByText('No audit entries')).toBeVisible({ timeout: 15000 })
  })
})
