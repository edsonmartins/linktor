import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

// Ao editar, a tela mostra "••••••••" no campo do segredo, prometendo que o
// valor guardado será mantido — e o backend faz exatamente isso (segredo em
// branco = manter). O schema, porém, exigia o campo, o que tornava impossível
// salvar qualquer alteração sem redigitar a credencial.
const casos = [
  { tipo: 'telegram',   nome: 'Bot de Suporte',   config: { bot_name: 'suporte_bot' } },
  { tipo: 'slack',      nome: 'Slack do Time',    config: { app_id: 'A123', bot_user_id: 'U123' } },
  { tipo: 'mattermost', nome: 'Mattermost Interno', config: { base_url: 'https://mm.exemplo.com', bot_user_id: 'b1' } },
]

for (const caso of casos) {
  test(`${caso.tipo}: salvar edição sem redigitar o segredo`, async ({ page }) => {
    const canal = {
      id: `ch-${caso.tipo}`,
      tenant_id: 't1',
      name: caso.nome,
      type: caso.tipo,
      enabled: true,
      connection_status: 'connected',
      config: caso.config,
      created_at: '2026-08-01T00:00:00Z',
      updated_at: '2026-08-01T00:00:00Z',
    }

    await setupAuth(page)
    let salvou = false
    await page.route('**/api/v1/channels**', (route) => {
      if (route.request().method() === 'PUT') {
        salvou = true
        return route.fulfill({ status: 200, contentType: 'application/json',
          body: JSON.stringify({ success: true, data: canal }) })
      }
      return route.fulfill({ status: 200, contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [canal], meta: { page: 1, page_size: 20, total_items: 1 } }) })
    })

    await page.goto('/channels')
    const card = page.getByRole('heading', { name: caso.nome })
      .locator('xpath=ancestor::div[contains(@class,"hover:border-primary/30")]')
    await card.locator('button[aria-haspopup="menu"]').click()
    await page.getByRole('menuitem', { name: /Configure|Configurar/i }).click()

    // Muda só o nome, deixando o segredo em branco — como a tela sugere.
    const campoNome = page.getByRole('dialog').getByRole('textbox').first()
    await expect(campoNome).toHaveValue(caso.nome) // veio preenchido
    await campoNome.fill(`${caso.nome} (renomeado)`)
    await page.getByRole('button', { name: /Update|Save|Atualizar|Salvar/i }).first().click()

    await expect.poll(() => salvou, { timeout: 6000 }).toBe(true)
  })
}

