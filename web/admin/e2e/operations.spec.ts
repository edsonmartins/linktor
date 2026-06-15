import { test, expect } from '@playwright/test'
import { setupAuth, mockEmptyEndpoints, mockAPIError } from './helpers'

type JsonRecord = Record<string, unknown>

const SETTINGS = {
  tenant_id: 'tenant-1',
  assignment_strategy: 'round_robin',
  auto_assign_enabled: true,
  sla_first_response_minutes: 15,
  sla_resolution_minutes: 240,
  auto_close_after_minutes: 1440,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

async function mockSettings(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/settings', async (route) => {
    if (route.request().method() === 'PUT') {
      const payload = route.request().postDataJSON() as JsonRecord
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { ...SETTINGS, ...payload } }),
      })
      return
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: SETTINGS }),
    })
  })
}

/**
 * Registers a PUT-capturing route on top of the beforeEach mock. GET requests
 * fall through to the generic mock so the form still hydrates.
 */
async function captureSave(page: import('@playwright/test').Page) {
  const captured: { payload: JsonRecord | null } = { payload: null }

  await page.route('**/api/v1/settings', async (route) => {
    if (route.request().method() === 'PUT') {
      captured.payload = route.request().postDataJSON() as JsonRecord
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: { ...SETTINGS, ...captured.payload },
        }),
      })
      return
    }

    await route.fallback()
  })

  return captured
}

/** Waits for the form to hydrate from GET /settings before interacting. */
async function waitForHydration(page: import('@playwright/test').Page) {
  await expect(page.getByRole('combobox')).toContainText('Round-robin', {
    timeout: 15000,
  })
  await expect(page.getByRole('spinbutton').first()).toHaveValue('15')
}

test.describe('Operations Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockEmptyEndpoints(page)
    await mockSettings(page)
  })

  test('loads assignment and SLA settings into the form', async ({ page }) => {
    await page.goto('/operations')

    await expect(page.getByRole('heading', { name: 'Operations' })).toBeVisible({
      timeout: 15000,
    })

    await expect(page.getByText('Assignment', { exact: true })).toBeVisible()
    await expect(page.getByText('SLA & auto-close')).toBeVisible()

    await expect(page.getByRole('combobox')).toContainText('Round-robin')
    await expect(page.getByLabel('Auto-assign new conversations')).toBeChecked()

    // SLA inputs in render order: first response, resolution, auto-close.
    const slaInputs = page.getByRole('spinbutton')
    await expect(slaInputs.nth(0)).toHaveValue('15')
    await expect(slaInputs.nth(1)).toHaveValue('240')
    await expect(slaInputs.nth(2)).toHaveValue('1440')
  })

  test('changes assignment strategy and saves', async ({ page }) => {
    const captured = await captureSave(page)

    await page.goto('/operations')
    await waitForHydration(page)

    await page.getByRole('combobox').click()
    await page.getByRole('option', { name: 'Load-balanced (fewest open)' }).click()

    await page.getByRole('button', { name: 'Save changes' }).click()

    await expect.poll(() => captured.payload).not.toBeNull()
    expect(captured.payload).toMatchObject({
      assignment_strategy: 'load_balanced',
      auto_assign_enabled: true,
      sla_first_response_minutes: 15,
      sla_resolution_minutes: 240,
      auto_close_after_minutes: 1440,
    })
  })

  test('toggles auto-assign off and saves', async ({ page }) => {
    const captured = await captureSave(page)

    await page.goto('/operations')
    await waitForHydration(page)

    const autoAssign = page.getByLabel('Auto-assign new conversations')
    await expect(autoAssign).toBeChecked()
    await autoAssign.click()
    await expect(autoAssign).not.toBeChecked()

    await page.getByRole('button', { name: 'Save changes' }).click()

    await expect.poll(() => captured.payload).not.toBeNull()
    expect(captured.payload).toMatchObject({
      assignment_strategy: 'round_robin',
      auto_assign_enabled: false,
    })
  })

  test('updates SLA values and saves', async ({ page }) => {
    const captured = await captureSave(page)

    await page.goto('/operations')
    await waitForHydration(page)

    const slaInputs = page.getByRole('spinbutton')
    await slaInputs.nth(0).fill('30')
    await slaInputs.nth(1).fill('480')
    await slaInputs.nth(2).fill('0')

    await page.getByRole('button', { name: 'Save changes' }).click()

    await expect.poll(() => captured.payload).not.toBeNull()
    expect(captured.payload).toMatchObject({
      sla_first_response_minutes: 30,
      sla_resolution_minutes: 480,
      auto_close_after_minutes: 0,
    })
  })

  test('shows success toast after saving', async ({ page }) => {
    await page.goto('/operations')
    await waitForHydration(page)

    await page.getByRole('button', { name: 'Save changes' }).click()

    await expect(page.getByText('Settings saved').first()).toBeVisible({ timeout: 15000 })
  })

  test('shows error toast when save fails', async ({ page }) => {
    // Fail every method first, then restore GET on top so the form still
    // hydrates; PUT falls through to the error route.
    await mockAPIError(page, 'settings', 500, 'Database unavailable')
    await page.route('**/api/v1/settings', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: SETTINGS }),
        })
        return
      }

      await route.fallback()
    })

    await page.goto('/operations')
    await waitForHydration(page)

    await page.getByRole('button', { name: 'Save changes' }).click()

    await expect(page.getByText('Database unavailable').first()).toBeVisible()
  })
})
