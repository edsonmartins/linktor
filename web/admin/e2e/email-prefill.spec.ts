import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

// Reproduz o caso relatado: abrir um canal de e-mail existente para editar.
// Antes, o formulário abria em branco — mudar só a criptografia exigia
// redigitar tudo, e o salvar era recusado por campos obrigatórios.
const canal = {
  id: 'ch-email-1',
  tenant_id: 't1',
  name: 'Alcada',
  type: 'email',
  enabled: true,
  connection_status: 'connected',
  config: {
    provider: 'smtp',
    from_name: 'Alcada',
    from_email: 'alcada@exemplo.mailgun.org',
    smtp_host: 'smtp.mailgun.org',
    smtp_port: '587',
    smtp_encryption: 'tls',
  },
  created_at: '2026-08-14T20:46:31Z',
  updated_at: '2026-08-14T20:57:00Z',
}

test.describe('Edição de canal de e-mail', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await page.route('**/api/v1/channels**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: [canal],
          meta: { page: 1, page_size: 20, total_items: 1 },
        }),
      })
    )
  })

  test('abre com os dados do canal preenchidos', async ({ page }) => {
    await page.goto('/channels')

    const card = page
      .getByRole('heading', { name: 'Alcada' })
      .locator('xpath=ancestor::div[contains(@class,"hover:border-primary/30")]')
    await card.locator('button[aria-haspopup="menu"]').click()
    await page.getByRole('menuitem', { name: /Configure|Configurar/i }).click()

    // Exatamente os campos que apareciam em branco:
    await expect(page.locator('#smtp_host')).toHaveValue('smtp.mailgun.org')
    await expect(page.locator('#smtp_port')).toHaveValue('587')
    await expect(page.locator('#from_email')).toHaveValue('alcada@exemplo.mailgun.org')
    await expect(page.locator('#name')).toHaveValue('Alcada')
  })
})
