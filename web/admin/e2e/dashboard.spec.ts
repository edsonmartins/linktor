import { test, expect } from '@playwright/test'
import { setupAuth, mockEmptyEndpoints } from './helpers'

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockEmptyEndpoints(page)
  })

  test('loads dashboard page', async ({ page }) => {
    await page.goto('/dashboard')
    await expect(page).toHaveURL(/dashboard/, { timeout: 15000 })
  })

  test('sidebar navigation is visible', async ({ page }) => {
    await page.goto('/dashboard')

    // Sidebar links use href-based navigation; pt-BR labels
    // Check that a link to /conversations exists
    await expect(page.locator('a[href="/conversations"]')).toBeVisible({ timeout: 15000 })
  })

  test('can navigate to channels page', async ({ page }) => {
    await page.goto('/dashboard')

    await page.locator('a[href="/channels"]').click({ timeout: 15000 })
    await expect(page).toHaveURL(/channels/, { timeout: 10000 })
  })

  test('can navigate to contacts page', async ({ page }) => {
    await page.goto('/dashboard')

    await page.locator('a[href="/contacts"]').click({ timeout: 15000 })
    await expect(page).toHaveURL(/contacts/, { timeout: 10000 })
  })
})
