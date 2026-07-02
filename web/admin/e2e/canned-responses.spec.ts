import { test, expect, type Page, type Route } from '@playwright/test'
import { setupAuth, mockEmptyEndpoints } from './helpers'

type JsonRecord = Record<string, unknown>

const CANNED = [
  {
    id: 'cr-1',
    tenant_id: 'tenant-1',
    shortcut: 'greeting',
    title: 'Warm greeting',
    content: 'Hello {{name}}, how can I help you today?',
    tags: ['sales', 'vip'],
    usage_count: 42,
    created_at: '2026-05-01T00:00:00Z',
    updated_at: '2026-05-02T00:00:00Z',
  },
  {
    id: 'cr-2',
    tenant_id: 'tenant-1',
    shortcut: 'farewell',
    title: 'Polite farewell',
    content: 'Thanks for reaching out. Have a great day!',
    tags: ['support'],
    usage_count: 7,
    created_at: '2026-05-01T00:00:00Z',
    updated_at: '2026-05-02T00:00:00Z',
  },
]

async function fulfillData(route: Route, data: unknown) {
  await route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({
      success: true,
      data,
      meta: {
        page: 1,
        page_size: 20,
        total_items: Array.isArray(data) ? data.length : 1,
      },
    }),
  })
}

async function mockCannedEndpoints(page: Page) {
  await mockEmptyEndpoints(page)

  // Generic handler — per-test capture routes are registered later and win.
  await page.route('**/api/v1/canned-responses**', async (route) => {
    const request = route.request()
    const method = request.method()

    if (method === 'GET') {
      const search = new URL(request.url()).searchParams.get('search')?.toLowerCase()
      const items = search
        ? CANNED.filter(
            (c) =>
              c.shortcut.toLowerCase().includes(search) ||
              c.title.toLowerCase().includes(search),
          )
        : CANNED
      await fulfillData(route, items)
      return
    }

    if (method === 'POST') {
      const payload = request.postDataJSON() as JsonRecord
      await fulfillData(route, { ...CANNED[0], id: 'cr-new', ...payload, usage_count: 0 })
      return
    }

    if (method === 'PUT') {
      const payload = request.postDataJSON() as JsonRecord
      await fulfillData(route, { ...CANNED[0], ...payload })
      return
    }

    // DELETE
    await fulfillData(route, null)
  })
}

test.describe('Canned Responses', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockCannedEndpoints(page)
  })

  test('lists quick replies and cancels a delete safely', async ({ page }) => {
    let deleteFired = false
    page.on('request', (req) => {
      if (req.url().includes('/api/v1/canned-responses') && req.method() === 'DELETE') {
        deleteFired = true
      }
    })

    await page.goto('/canned-responses')

    await expect(
      page.getByRole('heading', { name: 'Quick Replies' }),
    ).toBeVisible({ timeout: 15000 })

    await expect(page.getByText('/greeting')).toBeVisible()
    await expect(page.getByText('Warm greeting')).toBeVisible()
    await expect(page.getByText('vip')).toBeVisible()
    await expect(page.getByText('42', { exact: true })).toBeVisible()
    await expect(page.getByText('/farewell')).toBeVisible()

    // Open the delete confirmation, then cancel — nothing must be deleted
    const row = page.getByRole('row').filter({ hasText: '/greeting' })
    await row.getByRole('button').nth(1).click()

    const dialog = page.getByRole('alertdialog')
    await expect(dialog.getByText('Delete this quick reply?')).toBeVisible()
    await dialog.getByRole('button', { name: 'Cancel' }).click()

    await expect(dialog).toBeHidden()
    await expect(page.getByText('/greeting')).toBeVisible()
    expect(deleteFired).toBe(false)
  })

  test('searches by shortcut', async ({ page }) => {
    const urls: string[] = []
    page.on('request', (req) => {
      if (req.url().includes('/api/v1/canned-responses') && req.method() === 'GET') {
        urls.push(req.url())
      }
    })

    await page.goto('/canned-responses')
    await expect(page.getByText('/greeting')).toBeVisible({ timeout: 15000 })

    await page.getByPlaceholder('Search').fill('fare')

    await expect.poll(() => urls.some((u) => u.includes('search=fare'))).toBe(true)
    await expect(page.getByText('/farewell')).toBeVisible()
    await expect(page.getByText('/greeting')).toBeHidden()
  })

  test('creates a quick reply', async ({ page }) => {
    let createPayload: JsonRecord | null = null
    await page.route('**/api/v1/canned-responses', async (route) => {
      if (route.request().method() !== 'POST') {
        await route.fallback()
        return
      }
      const payload = route.request().postDataJSON() as JsonRecord
      createPayload = payload
      await fulfillData(route, { id: 'cr-new', tenant_id: 'tenant-1', usage_count: 0, ...payload })
    })

    await page.goto('/canned-responses')
    await expect(
      page.getByRole('heading', { name: 'Quick Replies' }),
    ).toBeVisible({ timeout: 15000 })

    await page.getByRole('button', { name: 'New reply' }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog.getByText('New quick reply')).toBeVisible()

    // The dialog labels are not programmatically associated with the inputs,
    // so address the four textboxes by position: shortcut, title, content, tags.
    const boxes = dialog.getByRole('textbox')
    await boxes.nth(0).fill('hello')
    await boxes.nth(1).fill('Quick hello')
    await boxes.nth(2).fill('Hello! How can I help?')
    await boxes.nth(3).fill('sales, onboarding')

    await dialog.getByRole('button', { name: 'Create', exact: true }).click()

    await expect.poll(() => createPayload).not.toBeNull()
    expect(createPayload).toMatchObject({
      shortcut: 'hello',
      title: 'Quick hello',
      content: 'Hello! How can I help?',
      tags: ['sales', 'onboarding'],
    })

    await expect(dialog).toBeHidden()
  })

  test('edits an existing quick reply', async ({ page }) => {
    let putURL: string | null = null
    let putPayload: JsonRecord | null = null
    await page.route('**/api/v1/canned-responses/cr-1', async (route) => {
      if (route.request().method() !== 'PUT') {
        await route.fallback()
        return
      }
      putURL = route.request().url()
      putPayload = route.request().postDataJSON() as JsonRecord
      await fulfillData(route, { ...CANNED[0], ...putPayload })
    })

    await page.goto('/canned-responses')
    await expect(page.getByText('/greeting')).toBeVisible({ timeout: 15000 })

    const row = page.getByRole('row').filter({ hasText: '/greeting' })
    await row.getByRole('button').first().click()

    const dialog = page.getByRole('dialog')
    await expect(dialog.getByText('Edit quick reply')).toBeVisible()

    const boxes = dialog.getByRole('textbox')
    await expect(boxes.nth(0)).toHaveValue('greeting')
    await expect(boxes.nth(2)).toHaveValue('Hello {{name}}, how can I help you today?')

    await boxes.nth(2).fill('Hi {{name}}! What can I do for you?')
    await dialog.getByRole('button', { name: 'Save', exact: true }).click()

    await expect.poll(() => putURL).not.toBeNull()
    expect(putURL).toContain('/canned-responses/cr-1')
    expect(putPayload).toMatchObject({
      shortcut: 'greeting',
      title: 'Warm greeting',
      content: 'Hi {{name}}! What can I do for you?',
      tags: ['sales', 'vip'],
    })

    await expect(dialog).toBeHidden()
  })

  test('deletes a quick reply after confirmation', async ({ page }) => {
    let deleteURL: string | null = null
    await page.route('**/api/v1/canned-responses/cr-2', async (route) => {
      if (route.request().method() !== 'DELETE') {
        await route.fallback()
        return
      }
      deleteURL = route.request().url()
      await fulfillData(route, null)
    })

    await page.goto('/canned-responses')
    await expect(page.getByText('/farewell')).toBeVisible({ timeout: 15000 })

    const row = page.getByRole('row').filter({ hasText: '/farewell' })
    await row.getByRole('button').nth(1).click()

    const dialog = page.getByRole('alertdialog')
    await expect(dialog.getByText('Delete this quick reply?')).toBeVisible()
    await dialog.getByRole('button', { name: 'Delete', exact: true }).click()

    await expect.poll(() => deleteURL).not.toBeNull()
    expect(deleteURL).toContain('/canned-responses/cr-2')
    await expect(dialog).toBeHidden()
  })

  test('requires shortcut and content before creating', async ({ page }) => {
    await page.goto('/canned-responses')
    await expect(
      page.getByRole('heading', { name: 'Quick Replies' }),
    ).toBeVisible({ timeout: 15000 })

    await page.getByRole('button', { name: 'New reply' }).click()

    const dialog = page.getByRole('dialog')
    const createButton = dialog.getByRole('button', { name: 'Create', exact: true })
    await expect(createButton).toBeDisabled()

    const boxes = dialog.getByRole('textbox')
    await boxes.nth(0).fill('promo')
    await expect(createButton).toBeDisabled() // content still missing
    await boxes.nth(2).fill('Check out our promo!')
    await expect(createButton).toBeEnabled()
  })
})
