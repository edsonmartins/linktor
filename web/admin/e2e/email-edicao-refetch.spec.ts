import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

// O canal muda entre uma leitura e outra da listagem — aqui via
// connection_status, mas na prática qualquer campo serve (updated_at,
// last_echo_at). Isso basta para o React Query devolver um objeto NOVO, e o
// efeito de preenchimento dispara de novo em cima do que a pessoa digitou.
function canal(status: string) {
  return {
    id: 'ch-email-alcada',
    tenant_id: 't1',
    name: 'Alçada',
    type: 'email',
    enabled: true,
    connection_status: status,
    config: {
      provider: 'smtp',
      from_name: 'Alçada',
      from_email: 'alcadaedsonmartins@gmail.com',
      smtp_host: 'smtp.gmail.com',
      smtp_port: '587',
      smtp_encryption: 'starttls',
      imap_host: 'imap.gmail.com',
      imap_port: '993',
      imap_folder: 'INBOX',
      imap_poll_interval: '30',
    },
    created_at: '2026-08-14T20:46:31Z',
    updated_at: '2026-08-15T01:00:00Z',
  }
}

test('editar credencial sobrevive a uma revalidação da listagem', async ({ page }) => {
  await setupAuth(page)

  let leituras = 0
  const enviado: { valor: { credentials?: Record<string, string> } | null } = { valor: null }

  await page.route('**/api/v1/channels**', (route) => {
    if (route.request().method() === 'PUT') {
      enviado.valor = route.request().postDataJSON()
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: canal('connected') }),
      })
    }
    leituras++
    // A segunda leitura devolve o canal com estado diferente — é o gatilho.
    return route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: [canal(leituras > 1 ? 'connected' : 'disconnected')],
        meta: { page: 1, page_size: 20, total_items: 1 },
      }),
    })
  })

  // A listagem tem staleTime de 1 minuto: só depois disso o retorno do foco
  // revalida. O relógio falso evita esperar esse minuto no teste.
  await page.clock.install()
  await page.goto('/channels')
  const card = page
    .getByRole('heading', { name: 'Alçada' })
    .locator('xpath=ancestor::div[contains(@class,"hover:border-primary/30")]')
  await card.locator('button[aria-haspopup="menu"]').click()
  await page.getByRole('menuitem', { name: /Configure|Configurar/i }).click()

  await page.getByRole('tab', { name: /Receiving|Recebimento/i }).click()
  await page.locator('#imap_username').fill('alcadaedsonmartins@gmail.com')

  // Sair da janela e voltar é o que acontece de verdade ao copiar a senha de
  // app de outra aba: o React Query revalida ao recuperar o foco.
  await page.clock.fastForward('01:30')
  await page.evaluate(() => {
    // O React Query v5 escuta 'visibilitychange' no window.
    window.dispatchEvent(new Event('visibilitychange'))
  })
  await expect.poll(() => leituras, { timeout: 5000 }).toBeGreaterThan(1)

  // O que a pessoa digitou tem de continuar lá.
  await expect(page.locator('#imap_username')).toHaveValue('alcadaedsonmartins@gmail.com')

  await page.getByRole('button', { name: /Update|Atualizar|Save|Salvar/i }).first().click()
  await expect.poll(() => enviado.valor, { timeout: 6000 }).not.toBeNull()
  expect(enviado.valor?.credentials?.imap_username).toBe('alcadaedsonmartins@gmail.com')
})
