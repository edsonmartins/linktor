import { expect, test, type Page } from '@playwright/test'
import { mockEmptyEndpoints, setupAuth } from './helpers'

/**
 * WP-L (fase 0.2): leitura da trilha de auditoria filtrável por autor e
 * período (além de ação), com escopo de tenant e admin-only. Os eventos desta
 * capacidade (channel.create/update, sandbox_allowlist.*) mostram os campos
 * não-secretos em Detalhes — nenhum valor de credencial.
 */

const SANDBOX_AUDIT = [
  {
    id: 'log-ch',
    tenant_id: 'tenant-1',
    actor_id: 'user-1',
    actor_email: 'alice@acme.com',
    actor_name: 'Alice',
    action: 'channel.create',
    resource_type: 'channel',
    resource_id: 'ch-sb-0000-0000-0000-000000000000',
    changes: { type: 'whatsapp_official', environment: 'sandbox', credential_environment: 'sandbox', phone_number_id: '111222333' },
    ip_address: '10.0.0.9',
    created_at: '2026-07-22T10:00:00Z',
  },
]

async function mockAudit(page: Page, capturedUrls: string[]) {
  await mockEmptyEndpoints(page)
  await page.route('**/api/v1/audit-logs**', async (route) => {
    capturedUrls.push(route.request().url())
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: SANDBOX_AUDIT,
        meta: { page: 1, page_size: 50, total_pages: 1, total_items: 1, has_next: false, has_previous: false },
      }),
    })
  })
}

test.describe('Audit trail — sandbox capability (WP-L)', () => {
  test('actor and period filters are sent to the backend', async ({ page }) => {
    await setupAuth(page, 'admin')
    const urls: string[] = []
    await mockAudit(page, urls)
    await page.goto('/audit-logs')

    await page.getByPlaceholder(/Filtrar por autor|Filter by actor/i).fill('alice')
    await page.locator('input[type="date"]').first().fill('2026-07-01')

    await expect
      .poll(() => urls.some((u) => u.includes('actor=alice') && u.includes('start_date=2026-07-01')))
      .toBe(true)
  })

  test('channel.create details show the sandbox boundary fields, no secrets', async ({ page }) => {
    await setupAuth(page, 'admin')
    await mockAudit(page, [])
    await page.goto('/audit-logs')

    // The prioritized event and its non-secret change fields are visible.
    await expect(page.getByText('channel.create')).toBeVisible()
    await expect(page.getByText('environment=sandbox', { exact: true })).toBeVisible()
    await expect(page.getByText('credential_environment=sandbox', { exact: true })).toBeVisible()
    // No credential value ever appears.
    await expect(page.getByText(/access_token/i)).toHaveCount(0)
  })

  test('non-admin cannot reach the audit trail', async ({ page }) => {
    await setupAuth(page, 'agent')
    await mockEmptyEndpoints(page)
    await page.goto('/audit-logs')
    await expect(page.getByText(/Acesso negado|Access denied/i)).toBeVisible()
  })
})
