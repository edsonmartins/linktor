import { test, expect, type Page } from '@playwright/test'
import { setupAuth } from './helpers'

/**
 * WP-I (fase 0.2): environment no formulário de canal.
 * - Criação: environment selecionável, default production; sandbox exige
 *   declaração de credencial e lista de phone_number_ids (whatsapp_official).
 * - Edição: environment SOMENTE LEITURA com a razão visível, e o campo não é
 *   submetido no update.
 */

const sandboxChannel = {
  id: 'ch-sb',
  name: 'Homologação ACME',
  type: 'whatsapp_official',
  enabled: true,
  connection_status: 'disconnected',
  environment: 'sandbox',
  config: {
    phone_number_id: '111222333',
    sandbox_test_phone_number_ids: '111222333',
    verify_token: 'tok',
    api_version: 'v23.0',
  },
  created_at: '2026-07-20T10:00:00Z',
  updated_at: '2026-07-20T10:00:00Z',
}

async function mockChannelsApi(page: Page, channels: unknown[]) {
  await page.route('**/api/v1/channels', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: channels }),
      })
      return
    }
    await route.fulfill({
      status: 201,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: { ...sandboxChannel, id: 'ch-new' } }),
    })
  })
  // Rotas auxiliares que a whatsapp-config consulta quando em edição.
  for (const sub of ['payments', 'calls', 'ctwa', 'coexistence', 'templates']) {
    await page.route(`**/api/v1/channels/**/${sub}**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: {} }),
      })
    })
  }
  await page.route('**/api/v1/channels/ch-sb', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: sandboxChannel }),
    })
  })
}

async function openAddWhatsAppOfficial(page: Page) {
  const addChannelSection = page
    .getByRole('heading', { name: /Add New Channel|Adicionar Novo Canal/i })
    .locator('xpath=../..')
  await addChannelSection.getByRole('heading', { name: 'WhatsApp Business' }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  return dialog
}

test.describe('Channel form environment (WP-I)', () => {
  test('creation: selectable with production default; sandbox reveals required fields', async ({ page }) => {
    await setupAuth(page)
    await mockChannelsApi(page, [])
    await page.goto('/channels')

    const dialog = await openAddWhatsAppOfficial(page)
    // O formulário (com o campo de ambiente) vive na aba de setup manual.
    await dialog.getByRole('tab', { name: /Manual/i }).click()

    // Default production no select de ambiente.
    const envTrigger = dialog.getByRole('combobox').filter({ hasText: /Production|Produção/i })
    await expect(envTrigger).toBeVisible()

    // Campos de sandbox ausentes enquanto production.
    await expect(dialog.getByText(/phone_number_ids/i)).toHaveCount(0)

    // Seleciona sandbox → campos obrigatórios aparecem.
    await envTrigger.click()
    await page.getByRole('option', { name: /Sandbox/i }).click()
    await expect(dialog.getByText(/phone_number_ids/i).first()).toBeVisible()
    await expect(dialog.getByRole('switch')).toBeVisible()
  })

  test('edit: environment is read-only with the immutability reason, not submitted', async ({ page }) => {
    await setupAuth(page)
    await mockChannelsApi(page, [sandboxChannel])

    let updatePayload: Record<string, unknown> | null = null
    await page.route('**/api/v1/channels/ch-sb', async (route) => {
      if (route.request().method() === 'PUT') {
        updatePayload = route.request().postDataJSON()
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: sandboxChannel }),
        })
        return
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: sandboxChannel }),
      })
    })

    await page.goto('/channels')
    const card = page
      .getByRole('heading', { name: 'Homologação ACME' })
      .locator('xpath=ancestor::div[contains(@class,"hover:border-primary/30")]')
    await card.locator('button[aria-haspopup="menu"]').click()
    await page.getByRole('menuitem', { name: /Configure|Configurar/i }).click()
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()

    // Aba de setup manual (onde o formulário vive; em edição já é a default).
    await dialog.getByRole('tab', { name: /Manual/i }).click()

    // Read-only: badge + razão da imutabilidade; NENHUM select de ambiente.
    await expect(dialog.getByRole('status', { name: /sandbox/i })).toBeVisible()
    await expect(dialog.getByText(/imutável|immutable/i).first()).toBeVisible()
    await expect(
      dialog.getByRole('combobox').filter({ hasText: /Production|Produção|Sandbox/i })
    ).toHaveCount(0)

    // Submete o update e verifica que environment NÃO vai no payload.
    // access_token é obrigatório; em edição o placeholder é mascarado.
    await dialog.getByPlaceholder('••••••••••••••••').fill('new-test-token')
    await dialog.getByRole('button', { name: /Update Channel|Atualizar Canal/i }).click()
    await expect.poll(() => updatePayload).not.toBeNull()
    const payload = updatePayload as unknown as Record<string, unknown>
    expect(payload).not.toHaveProperty('environment')
    expect(payload.credentials).toMatchObject({ credential_environment: 'sandbox' })
  })
})
