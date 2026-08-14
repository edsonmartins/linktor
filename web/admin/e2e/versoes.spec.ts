import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

// Responde "subiu ou não" sem entrar na VPS — e avisa quando o deploy aplicou
// só metade (painel novo com API velha, ou o contrário).
test.describe('Janela de versões', () => {
  test('mostra as duas versões e confirma quando coincidem', async ({ page }) => {
    await setupAuth(page)
    await page.route('**/health', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json',
        body: JSON.stringify({ service: 'linktor', status: 'ok', version: 'sha-abc1234' }) })
    )

    await page.goto('/dashboard')
    await page.getByRole('button', { name: /Versões|Versions|Versiones/i }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog.getByText('sha-abc1234')).toBeVisible()
  })

  test('avisa quando painel e API estão em versões diferentes', async ({ page }) => {
    await setupAuth(page)
    await page.route('**/health', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json',
        body: JSON.stringify({ service: 'linktor', status: 'ok', version: 'sha-diferente' }) })
    )

    await page.goto('/dashboard')
    await page.getByRole('button', { name: /Versões|Versions|Versiones/i }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog.getByText('sha-diferente')).toBeVisible()
  })

  test('API fora do ar não quebra a janela', async ({ page }) => {
    await setupAuth(page)
    await page.route('**/health', (route) => route.fulfill({ status: 500 }))

    await page.goto('/dashboard')
    await page.getByRole('button', { name: /Versões|Versions|Versiones/i }).click()

    await expect(page.getByRole('dialog')).toBeVisible()
    await expect(page.getByRole('dialog').getByText(/no response|sem resposta|sin respuesta/i)).toBeVisible()
  })
})
