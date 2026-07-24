import { test, expect, type Page } from '@playwright/test'
import { setupAuth } from './helpers'

/**
 * WP-J (fase 0.2): CRUD da allowlist de sandbox no console (INV-017, INV-023).
 * - Admin-only: rota gated; não-admin vê acesso negado (a API também recusa).
 * - Valor normalizado em E.164 exibido após a gravação (não o digitado).
 * - Confirmação de remoção explicita efeito no PRÓXIMO envio.
 * - "Sem allowlist configurada" (lista vazia) distinta de erro de carga.
 */

async function mockChannelsEmpty(page: Page) {
  await page.route('**/api/v1/channels', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: [] }),
    })
  })
}

test.describe('Sandbox allowlist (WP-J)', () => {
  test('non-admin is denied the route', async ({ page }) => {
    await setupAuth(page, 'agent')
    await mockChannelsEmpty(page)
    await page.goto('/sandbox-allowlist')
    // RoleGuard renders the access-denied screen; the write form never shows.
    await expect(page.getByText(/Acesso negado|Access denied/i)).toBeVisible()
    await expect(page.getByRole('button', { name: /Adicionar|Add recipient/i })).toHaveCount(0)
  })

  test('empty list is shown distinctly from a load error', async ({ page }) => {
    await setupAuth(page, 'admin')
    await mockChannelsEmpty(page)
    await page.route('**/api/v1/sandbox/allowlist', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      })
    })
    await page.goto('/sandbox-allowlist')
    // "no allowlist configured" — read succeeded, list empty.
    await expect(page.getByText(/Nenhum destinatário|No recipients/i)).toBeVisible()
    await expect(page.getByText(/TODO envio sandbox|EVERY sandbox send/i)).toBeVisible()
  })

  test('add shows the backend-normalized E.164 value, not what was typed', async ({ page }) => {
    await setupAuth(page, 'admin')
    await mockChannelsEmpty(page)
    let listData: unknown[] = []
    await page.route('**/api/v1/sandbox/allowlist', async (route) => {
      if (route.request().method() === 'POST') {
        // The user typed a "messy" number; the backend returns normalized E.164.
        const created = {
          id: 'e1',
          tenant_id: 'tenant-1',
          recipient: '+5544999999999',
          created_at: '2026-07-23T10:00:00Z',
        }
        listData = [created]
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: created }),
        })
        return
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: listData }),
      })
    })

    await page.goto('/sandbox-allowlist')
    // Target by stable id (locale-independent): the app may render in en/pt/es.
    await page.locator('#sb-recipient').fill('+55 44 9.9999-9999')
    await page.getByRole('button', { name: /Adicionar destinatário|Add recipient/i }).click()

    // The normalized value appears in the TABLE cell (not the typed formatting).
    await expect(page.getByRole('cell', { name: '+5544999999999' })).toBeVisible()
  })

  test('removal confirmation explains the immediate effect', async ({ page }) => {
    await setupAuth(page, 'admin')
    await mockChannelsEmpty(page)
    await page.route('**/api/v1/sandbox/allowlist', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: [
            {
              id: 'e1',
              tenant_id: 'tenant-1',
              recipient: '+5544999999999',
              created_at: '2026-07-23T10:00:00Z',
            },
          ],
        }),
      })
    })

    await page.goto('/sandbox-allowlist')
    await page.getByRole('button', { name: /Remover|Remove/i }).click()
    const dialog = page.getByRole('alertdialog')
    await expect(dialog).toBeVisible()
    // The confirmation must state the next-send effect (não é config).
    await expect(dialog.getByText(/PRÓXIMO envio|NEXT send/i)).toBeVisible()
  })
})
