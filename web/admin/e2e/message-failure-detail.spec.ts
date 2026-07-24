import { test, expect, type Page } from '@playwright/test'
import { setupAuth } from './helpers'

/**
 * WP-K (fase 0.2): diagnóstico de bloqueio na timeline.
 * - Bloqueio local de guarda (metadata.blocked_by) distinguível de falha de
 *   provedor, com motivo acionável e valor técnico disponível.
 * - Motivo desconhecido: valor bruto apresentado, nunca adivinhado.
 * Classificação vem do backend (INV-014): sem inferência no cliente.
 */

const conv = {
  id: 'conv-1',
  channel_id: 'ch-1',
  contact_id: 'contact-1',
  environment: 'sandbox',
  status: 'open',
  priority: 'normal',
  unread_count: 0,
  created_at: '2026-07-20T10:00:00Z',
  updated_at: '2026-07-20T12:00:00Z',
  last_message_at: '2026-07-20T12:00:00Z',
  contact: { id: 'contact-1', name: 'Homologação ACME', phone: '+5511999999999' },
  channel: { id: 'ch-1', type: 'whatsapp_official', environment: 'sandbox' },
}

function failedMessage(id: string, blockedBy: string | null, errorMessage: string) {
  return {
    id,
    conversation_id: 'conv-1',
    sender_type: 'user',
    sender_id: 'user-1', // matches the mock user → isOwn
    content_type: 'text',
    content: 'test',
    status: 'failed',
    error_message: errorMessage,
    metadata: blockedBy ? { blocked_by: blockedBy } : {},
    created_at: '2026-07-20T12:00:00Z',
    updated_at: '2026-07-20T12:00:00Z',
  }
}

async function mockConversation(page: Page, messages: unknown[]) {
  await page.route('**/api/v1/conversations**', async (route) => {
    const url = route.request().url()
    if (url.includes('/messages')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: messages }),
      })
      return
    }
    if (/\/conversations\/conv-1(\?|$)/.test(url)) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: conv }),
      })
      return
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: [conv] }),
    })
  })
  // Silence the auxiliary calls chat-view makes on mount so the test is
  // hermetic (no live requests to a non-running backend).
  await page.route('**/api/v1/conversations/conv-1/escalation-context', async (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ success: true, data: {} }) })
  )
  await page.route('**/api/v1/users**', async (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ success: true, data: [] }) })
  )
}

test.describe('Message failure diagnosis (WP-K)', () => {
  test('local guard block is distinguished from a provider rejection', async ({ page }) => {
    await setupAuth(page, 'admin')
    await mockConversation(page, [
      failedMessage('m-block', 'allowlist', 'sandbox guard: recipient +55*******99 is not in the tenant\'s sandbox allowlist'),
      failedMessage('m-provider', null, 'provider 470: message failed to send'),
    ])

    await page.goto('/conversations')
    await page.getByRole('button').filter({ hasText: 'Homologação ACME' }).first().click()

    // Local block: "Blocked locally" + the ACTIONABLE guidance. Assert on the
    // fix verb phrase ("Add it under" / "Adicione-o" / "Agréguelo"), which
    // appears ONLY in the translated reason paragraph — never in the raw
    // error_message nor in the sidebar nav link — so this cannot pass on the
    // technical-detail line if the reason rendering regresses.
    await expect(page.getByText(/Bloqueado localmente|Blocked locally/i)).toBeVisible()
    await expect(
      page.getByText(/Adicione-o|Add it under|Agréguelo/i)
    ).toBeVisible()

    // Provider failure: distinct "Rejected by the provider" (pt/en/es).
    await expect(
      page.getByText(/Rejeitado pelo provedor|Rejected by the provider|Rechazado por el proveedor/i)
    ).toBeVisible()

    // The technical detail is available for the block, with masked number.
    await expect(page.getByText(/\+55\*+99/)).toBeVisible()
  })

  test('unknown block reason shows the raw value, not a guess', async ({ page }) => {
    await setupAuth(page, 'admin')
    await mockConversation(page, [
      failedMessage('m-x', 'some_future_reason', 'blocked'),
    ])

    await page.goto('/conversations')
    await page.getByRole('button').filter({ hasText: 'Homologação ACME' }).first().click()

    // Still classified as a local block, and the raw reason is shown verbatim
    // (localBlock text is identical in pt/es; the regex also covers en).
    await expect(page.getByText(/Bloqueado localmente|Blocked locally/i)).toBeVisible()
    await expect(page.getByText(/some_future_reason/)).toBeVisible()
  })
})
