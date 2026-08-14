import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

// Canal de e-mail gravado com STARTTLS — o caso que quebrava: o seletor abria
// mostrando "TLS", e salvar qualquer outra coisa (usuário, senha) devolvia o
// canal para TLS sem que ninguém tivesse mexido na criptografia.
const canal = {
  id: 'ch-email-alcada',
  tenant_id: 't1',
  name: 'Alcada',
  type: 'email',
  enabled: true,
  connection_status: 'connected',
  config: {
    provider: 'smtp',
    from_name: 'Alcada',
    from_email: 'alcada@exemplo.org',
    smtp_host: 'smtp.mailgun.org',
    smtp_port: '587',
    smtp_encryption: 'starttls',
  },
  created_at: '2026-08-14T20:46:31Z',
  updated_at: '2026-08-14T20:57:00Z',
}

type Enviado = { config?: Record<string, string> } | null

/** Intercepta a listagem e captura o corpo do PUT de atualização. */
async function abrirConfiguracao(page: import('@playwright/test').Page) {
  const capturado: { valor: Enviado } = { valor: null }

  await page.route('**/api/v1/channels**', (route) => {
    if (route.request().method() === 'PUT') {
      capturado.valor = route.request().postDataJSON()
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: canal }),
      })
    }
    return route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: [canal],
        meta: { page: 1, page_size: 20, total_items: 1 },
      }),
    })
  })

  await page.goto('/channels')
  const card = page
    .getByRole('heading', { name: 'Alcada' })
    .locator('xpath=ancestor::div[contains(@class,"hover:border-primary/30")]')
  await card.locator('button[aria-haspopup="menu"]').click()
  await page.getByRole('menuitem', { name: /Configure|Configurar/i }).click()

  return capturado
}

test('o seletor abre exibindo a criptografia gravada', async ({ page }) => {
  await setupAuth(page)
  await abrirConfiguracao(page)

  // Compara o conjunto de seletores, e não um índice: se o valor não aparecer,
  // a falha mostra o que ESTAVA na tela em vez de estourar por timeout.
  await expect
    .poll(async () => (await page.getByRole('combobox').allTextContents()).join(' | '))
    .toContain('STARTTLS')
})

test('alterar só as credenciais preserva a criptografia gravada', async ({ page }) => {
  await setupAuth(page)
  const capturado = await abrirConfiguracao(page)

  await page.locator('#smtp_username').fill('postmaster@alcada.org')
  await page.locator('#smtp_password').fill('senha-nova')
  await page.getByRole('button', { name: /Update|Atualizar|Save|Salvar/i }).first().click()

  await expect.poll(() => capturado.valor, { timeout: 6000 }).not.toBeNull()
  expect(capturado.valor?.config?.smtp_encryption).toBe('starttls')
})

test('trocar a criptografia chega no PUT', async ({ page }) => {
  await setupAuth(page)
  const capturado = await abrirConfiguracao(page)

  const seletor = page.getByRole('combobox').filter({ hasText: /TLS|STARTTLS/i }).first()
  await seletor.click()
  await page.getByRole('option', { name: /^TLS/ }).click()
  await page.getByRole('button', { name: /Update|Atualizar|Save|Salvar/i }).first().click()

  await expect.poll(() => capturado.valor, { timeout: 6000 }).not.toBeNull()
  expect(capturado.valor?.config?.smtp_encryption).toBe('tls')
})
