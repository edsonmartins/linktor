import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

/**
 * WP-H (fase 0.2): marcação visual de ambiente (INV-018 / proposta INV-025).
 * - Conversa sandbox listada com rótulo TEXTUAL (não só cor), valor vindo da API.
 * - Filtro de ambiente aplicado no backend (query param), não em memória.
 * - Conversa production não muda de apresentação.
 */

function conversation(id: string, environment: 'production' | 'sandbox', name: string) {
  return {
    id,
    channel_id: 'ch-1',
    contact_id: `contact-${id}`,
    environment,
    status: 'open',
    priority: 'normal',
    unread_count: 0,
    created_at: '2026-07-20T10:00:00Z',
    updated_at: '2026-07-20T12:00:00Z',
    last_message_at: '2026-07-20T12:00:00Z',
    contact: { id: `contact-${id}`, name, phone: '+5511999999999' },
    channel: { id: 'ch-1', type: 'whatsapp_official', environment },
    last_message: { content: 'olá', created_at: '2026-07-20T12:00:00Z' },
  }
}

test.describe('Sandbox environment marking', () => {
  test('sandbox conversation shows textual badge from API value', async ({ page }) => {
    await setupAuth(page)
    await page.route('**/api/v1/conversations**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: [
            conversation('conv-sb', 'sandbox', 'Homologação ACME'),
            conversation('conv-prod', 'production', 'Cliente Real'),
          ],
        }),
      })
    })

    await page.goto('/conversations')

    const sandboxItem = page.getByRole('button').filter({ hasText: 'Homologação ACME' })
    // Rótulo textual, acessível (role=status), não apenas cor:
    await expect(sandboxItem.getByRole('status', { name: /sandbox/i })).toBeVisible()
    await expect(sandboxItem.getByText('Sandbox', { exact: false })).toBeVisible()

    // Conversa production: apresentação inalterada — nenhum badge de ambiente.
    const prodItem = page.getByRole('button').filter({ hasText: 'Cliente Real' })
    await expect(prodItem.getByText('Sandbox', { exact: false })).toHaveCount(0)
  })

  test('environment filter is sent to the backend as query param', async ({ page }) => {
    await setupAuth(page)
    const requestedUrls: string[] = []
    await page.route('**/api/v1/conversations**', async (route) => {
      requestedUrls.push(route.request().url())
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      })
    })

    await page.goto('/conversations')
    // Abre o menu de filtros e escolhe Sandbox.
    await page.getByRole('button').filter({ has: page.locator('svg.lucide-filter') }).click()
    await page.getByRole('menuitem', { name: 'Sandbox' }).click()

    await expect
      .poll(() => requestedUrls.some((u) => u.includes('environment=sandbox')))
      .toBe(true)
  })
})
