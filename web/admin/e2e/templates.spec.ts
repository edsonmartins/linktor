import { test, expect, type Page, type Route } from '@playwright/test'
import { setupAuth, mockEmptyEndpoints, mockList } from './helpers'

type JsonRecord = Record<string, unknown>

const CHANNEL = {
  id: 'ch-1',
  tenant_id: 'tenant-1',
  name: 'WhatsApp Business',
  type: 'whatsapp_official',
  enabled: true,
  connection_status: 'connected',
  config: {},
  created_at: '2026-05-01T00:00:00Z',
  updated_at: '2026-05-01T00:00:00Z',
}

const TPL_APPROVED = {
  id: 'tpl-1',
  tenant_id: 'tenant-1',
  channel_id: 'ch-1',
  external_id: 'meta-1001',
  name: 'order_confirmation',
  language: 'pt_BR',
  category: 'UTILITY',
  status: 'APPROVED',
  quality_score: 'GREEN',
  components: [
    {
      type: 'BODY',
      text: 'Hi {{1}}, your order {{2}} has been confirmed!',
      example: { body_text: [['Maria', '#1234']] },
    },
    { type: 'FOOTER', text: 'Linktor' },
    { type: 'BUTTONS', buttons: [{ type: 'QUICK_REPLY', text: 'Track order' }] },
  ],
  last_synced_at: '2026-05-02T10:00:00Z',
  created_at: '2026-05-01T00:00:00Z',
  updated_at: '2026-05-02T00:00:00Z',
}

const TPL_REJECTED = {
  id: 'tpl-2',
  tenant_id: 'tenant-1',
  channel_id: 'ch-1',
  external_id: 'meta-1002',
  name: 'promo_blast',
  language: 'pt_BR',
  category: 'MARKETING',
  status: 'REJECTED',
  quality_score: 'UNKNOWN',
  rejection_reason: 'Policy violation',
  components: [{ type: 'BODY', text: 'Big sale today!' }],
  created_at: '2026-05-01T00:00:00Z',
  updated_at: '2026-05-02T00:00:00Z',
}

const TPL_PENDING = {
  id: 'tpl-3',
  tenant_id: 'tenant-1',
  channel_id: 'ch-1',
  external_id: '',
  name: 'auth_code',
  language: 'en_US',
  category: 'AUTHENTICATION',
  status: 'PENDING',
  quality_score: 'UNKNOWN',
  components: [{ type: 'BODY', text: 'Your code is {{1}}.', example: { body_text: [['123456']] } }],
  created_at: '2026-05-01T00:00:00Z',
  updated_at: '2026-05-02T00:00:00Z',
}

const ALL_TEMPLATES = [TPL_APPROVED, TPL_REJECTED, TPL_PENDING]

const LIBRARY_ITEMS = [
  {
    name: 'order_shipped_meta',
    category: 'UTILITY',
    language: 'en_US',
    topic: 'ORDER_UPDATE',
    usecase: 'shipping',
    body_text: 'Your order {{1}} has shipped.',
  },
  {
    name: 'payment_reminder_meta',
    category: 'UTILITY',
    language: 'en_US',
    topic: 'PAYMENT_REMINDER',
    usecase: 'billing',
    body_text: 'Your payment of {{1}} is due on {{2}}.',
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

async function mockTemplateEndpoints(page: Page) {
  await mockEmptyEndpoints(page)
  await mockList(page, 'channels', [CHANNEL])

  // Generic /templates handler. Registered FIRST so the more specific routes
  // below (and per-test capture routes) take precedence — Playwright matches
  // the most recently registered route first.
  await page.route('**/api/v1/templates**', async (route) => {
    const request = route.request()
    const method = request.method()
    const url = new URL(request.url())

    if (method === 'POST') {
      if (url.pathname.includes('/sync/')) {
        await fulfillData(route, null)
        return
      }
      if (url.pathname.endsWith('/refresh')) {
        await fulfillData(route, TPL_APPROVED)
        return
      }
      const payload = request.postDataJSON() as JsonRecord
      await fulfillData(route, {
        ...TPL_APPROVED,
        id: 'tpl-new',
        name: payload.name,
        language: payload.language,
        status: 'PENDING',
        external_id: '',
      })
      return
    }

    if (method === 'DELETE') {
      await fulfillData(route, null)
      return
    }

    if (url.pathname.endsWith('/api/v1/templates')) {
      // List, honoring the filters the page sends.
      const name = url.searchParams.get('name')
      const status = url.searchParams.get('status')
      const category = url.searchParams.get('category')
      let items = ALL_TEMPLATES
      if (name) items = items.filter((t) => t.name.includes(name))
      if (status) items = items.filter((t) => t.status === status)
      if (category) items = items.filter((t) => t.category === category)
      await fulfillData(route, items)
      return
    }

    // Detail GET for any id (known fixtures or templates "created" in a test).
    const id = url.pathname.split('/').pop() ?? ''
    const found = ALL_TEMPLATES.find((t) => t.id === id)
    await fulfillData(route, found ?? { ...TPL_APPROVED, id })
  })

  // Library endpoints — more specific, so registered AFTER the generic route.
  await page.route('**/api/v1/templates/library**', async (route) => {
    if (route.request().method() === 'POST') {
      const payload = route.request().postDataJSON() as JsonRecord
      await fulfillData(route, {
        ...TPL_APPROVED,
        id: 'tpl-lib',
        name: payload.name,
        language: payload.language,
      })
      return
    }
    await fulfillData(route, LIBRARY_ITEMS)
  })
}

test.describe('Templates', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockTemplateEndpoints(page)
  })

  test('lists templates with status badges and searches by name', async ({ page }) => {
    await page.goto('/templates')

    await expect(
      page.getByRole('heading', { name: 'WhatsApp Templates' }),
    ).toBeVisible({ timeout: 15000 })

    await expect(page.getByText('order_confirmation')).toBeVisible()
    await expect(page.getByText('promo_blast')).toBeVisible()
    await expect(page.getByText('auth_code')).toBeVisible()

    // Status badges
    await expect(page.getByText('Approved', { exact: true })).toBeVisible()
    await expect(page.getByText('Rejected', { exact: true })).toBeVisible()
    await expect(page.getByText('Pending', { exact: true })).toBeVisible()

    // Category labels rendered per row
    await expect(page.getByText('Marketing', { exact: true })).toBeVisible()
    await expect(page.getByText('Authentication', { exact: true })).toBeVisible()

    // Search narrows the list via the `name` filter
    await page.getByPlaceholder('Search by name…').fill('promo')
    await expect(page.getByText('promo_blast')).toBeVisible()
    await expect(page.getByText('order_confirmation')).toBeHidden()
  })

  test('filters by status', async ({ page }) => {
    const listURLs: string[] = []
    page.on('request', (req) => {
      if (req.url().includes('/api/v1/templates') && req.method() === 'GET') {
        listURLs.push(req.url())
      }
    })

    await page.goto('/templates')
    await expect(page.getByText('order_confirmation')).toBeVisible({ timeout: 15000 })

    await page.getByRole('combobox').filter({ hasText: 'All statuses' }).click()
    await page.getByRole('option', { name: 'Approved' }).click()

    await expect.poll(() => listURLs.some((u) => u.includes('status=APPROVED'))).toBe(true)
    await expect(page.getByText('order_confirmation')).toBeVisible()
    await expect(page.getByText('promo_blast')).toBeHidden()
  })

  test('navigates to the new template form', async ({ page }) => {
    await page.goto('/templates')
    await expect(
      page.getByRole('heading', { name: 'WhatsApp Templates' }),
    ).toBeVisible({ timeout: 15000 })

    await page.getByRole('button', { name: 'New template' }).click()

    // Generous timeout: the dev server compiles the route on first visit.
    await expect(page).toHaveURL(/\/templates\/new/, { timeout: 15000 })
    await expect(
      page.getByRole('heading', { name: 'New template' }),
    ).toBeVisible({ timeout: 15000 })
    await expect(page.getByLabel('Technical name')).toBeVisible()
  })

  test('creates a template and submits the right payload', async ({ page }) => {
    let createPayload: JsonRecord | null = null
    await page.route('**/api/v1/templates', async (route) => {
      if (route.request().method() !== 'POST') {
        await route.fallback()
        return
      }
      const payload = route.request().postDataJSON() as JsonRecord
      createPayload = payload
      await fulfillData(route, {
        ...TPL_APPROVED,
        id: 'tpl-new',
        name: payload.name,
        language: payload.language,
        status: 'PENDING',
        external_id: '',
      })
    })

    await page.goto('/templates/new')
    await expect(
      page.getByRole('heading', { name: 'New template' }),
    ).toBeVisible({ timeout: 15000 })

    // Channel select (shadcn Select — opens on click, options have role=option)
    await page.getByRole('combobox', { name: 'Channel' }).click()
    await page.getByRole('option', { name: 'WhatsApp Business' }).click()

    await page.getByLabel('Technical name').fill('order_update')
    await page.getByLabel('Language').fill('en_US')
    await page.getByLabel('Text', { exact: true }).fill('Your order has shipped and is on its way.')
    await page.getByLabel('Footer').fill('Linktor')

    // Add a quick-reply button
    await page.getByRole('button', { name: 'Quick reply' }).click()
    await page.getByPlaceholder('Button text (25 chars max)').fill('Track order')

    await page.getByRole('button', { name: 'Submit for approval' }).click()

    await expect.poll(() => createPayload).not.toBeNull()
    expect(createPayload).toMatchObject({
      channel_id: 'ch-1',
      name: 'order_update',
      language: 'en_US',
      category: 'UTILITY',
      parameter_format: 'POSITIONAL',
      components: [
        { type: 'BODY', text: 'Your order has shipped and is on its way.' },
        { type: 'FOOTER', text: 'Linktor' },
        { type: 'BUTTONS', buttons: [{ type: 'QUICK_REPLY', text: 'Track order' }] },
      ],
    })

    // On success the form navigates to the new template's detail page
    // (generous timeout: the detail route compiles on demand in dev).
    await expect(page).toHaveURL(/\/templates\/tpl-new/, { timeout: 15000 })
  })

  test('shows template detail and refreshes from Meta', async ({ page }) => {
    let refreshed = false
    await page.route('**/api/v1/templates/tpl-1/refresh', async (route) => {
      refreshed = true
      await fulfillData(route, TPL_APPROVED)
    })

    await page.goto('/templates/tpl-1')

    await expect(
      page.getByRole('heading', { name: 'order_confirmation' }),
    ).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('Approved', { exact: true })).toBeVisible()
    // Preview renders the BODY text and the button row
    await expect(
      page.getByText('Hi {{1}}, your order {{2}} has been confirmed!'),
    ).toBeVisible()
    await expect(page.getByText('Track order')).toBeVisible()
    // Metadata card
    await expect(page.getByText('meta-1001')).toBeVisible()

    await page.getByRole('button', { name: 'Refresh' }).click()
    await expect.poll(() => refreshed).toBe(true)
  })

  test('deletes a template from the detail page after confirmation', async ({ page }) => {
    let deleteRequested = false
    await page.route('**/api/v1/templates/tpl-1', async (route) => {
      if (route.request().method() !== 'DELETE') {
        await route.fallback()
        return
      }
      deleteRequested = true
      await fulfillData(route, null)
    })

    await page.goto('/templates/tpl-1')
    await expect(
      page.getByRole('heading', { name: 'order_confirmation' }),
    ).toBeVisible({ timeout: 15000 })

    await page.getByRole('button', { name: 'Delete' }).click()

    const dialog = page.getByRole('alertdialog')
    await expect(dialog.getByText('Delete template?')).toBeVisible()
    await dialog.getByRole('button', { name: 'Delete' }).click()

    await expect.poll(() => deleteRequested).toBe(true)
    await expect(page).toHaveURL(/\/templates$/)
  })

  test('deletes a template from the list', async ({ page }) => {
    let deletedURL: string | null = null
    await page.route('**/api/v1/templates/tpl-2', async (route) => {
      if (route.request().method() !== 'DELETE') {
        await route.fallback()
        return
      }
      deletedURL = route.request().url()
      await fulfillData(route, null)
    })

    await page.goto('/templates')
    await expect(page.getByText('promo_blast')).toBeVisible({ timeout: 15000 })

    // The trash button only becomes visible on row hover
    const row = page.locator('div.group', { hasText: 'promo_blast' })
    await row.hover()
    await row.getByRole('button').click()

    const dialog = page.getByRole('alertdialog')
    await expect(dialog.getByText('Delete template?')).toBeVisible()
    await dialog.getByRole('button', { name: 'Delete' }).click()

    await expect.poll(() => deletedURL).not.toBeNull()
    expect(deletedURL).toContain('/templates/tpl-2')
  })

  test('browses the Meta library and instantiates a template', async ({ page }) => {
    let libPayload: JsonRecord | null = null
    await page.route('**/api/v1/templates/library', async (route) => {
      if (route.request().method() !== 'POST') {
        await route.fallback()
        return
      }
      const payload = route.request().postDataJSON() as JsonRecord
      libPayload = payload
      await fulfillData(route, { ...TPL_APPROVED, id: 'tpl-lib', name: payload.name })
    })

    await page.goto('/templates/library')

    await expect(
      page.getByRole('heading', { name: 'Meta library' }),
    ).toBeVisible({ timeout: 15000 })

    // Single WhatsApp channel is auto-selected, so the catalog loads directly
    await expect(page.getByText('order_shipped_meta')).toBeVisible()
    await expect(page.getByText('payment_reminder_meta')).toBeVisible()

    await page.getByRole('button', { name: 'Use this template' }).first().click()

    const dialog = page.getByRole('dialog')
    await expect(dialog.getByText('Instantiate library template')).toBeVisible()
    // Name is prefilled from the library entry
    await expect(dialog.getByLabel('Technical name')).toHaveValue('order_shipped_meta')

    await dialog.getByLabel('Technical name').fill('my_shipping_update')
    await dialog.getByRole('button', { name: 'Create', exact: true }).click()

    await expect.poll(() => libPayload).not.toBeNull()
    expect(libPayload).toMatchObject({
      channel_id: 'ch-1',
      name: 'my_shipping_update',
      language: 'en_US',
      category: 'UTILITY',
      library_template_name: 'order_shipped_meta',
    })

    await expect(page).toHaveURL(/\/templates\/tpl-lib/)
  })
})
