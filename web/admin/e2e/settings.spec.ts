import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

async function mockSettingsEndpoints(page: import('@playwright/test').Page) {
  for (const endpoint of ['conversations', 'channels', 'contacts', 'bots', 'flows', 'knowledge-bases']) {
    await page.route(`**/api/v1/${endpoint}**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      })
    })
  }
}

test.describe('Settings Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockSettingsEndpoints(page)
  })

  test('renders settings page', async ({ page }) => {
    await page.goto('/settings')

    await expect(page.getByText('Settings').first()).toBeVisible({ timeout: 15000 })
  })

  test('shows profile section', async ({ page }) => {
    await page.goto('/settings')

    await expect(page.getByText('Profile').first()).toBeVisible({ timeout: 15000 })

    // Profile section has name and email inputs
    await expect(page.locator('input#name')).toBeVisible()
    await expect(page.locator('input#email')).toBeVisible()
  })

  test('shows user role badge', async ({ page }) => {
    await page.goto('/settings')

    await expect(page.getByText('Profile').first()).toBeVisible({ timeout: 15000 })

    // Role badge should show "admin" from the mock user
    await expect(page.getByText('admin').first()).toBeVisible()
  })

  test('has navigation tabs', async ({ page }) => {
    await page.goto('/settings')

    await expect(page.getByText('Profile').first()).toBeVisible({ timeout: 15000 })

    // Settings navigation should have all section tabs
    await expect(page.getByText('Notifications').first()).toBeVisible()
    await expect(page.getByText('Security').first()).toBeVisible()
    await expect(page.getByText('Appearance').first()).toBeVisible()
    await expect(page.getByText('API Keys').first()).toBeVisible()
    await expect(page.getByText('Organization').first()).toBeVisible()
  })
})
