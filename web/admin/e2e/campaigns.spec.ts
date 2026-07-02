import { test, expect } from '@playwright/test'
import { setupAuth, mockEmptyEndpoints } from './helpers'

type JsonRecord = Record<string, unknown>

const CAMPAIGNS = [
  {
    id: 'camp-1',
    tenant_id: 'tenant-1',
    channel_id: 'ch-1',
    name: 'April Promo',
    template_name: 'promo_april',
    template_language: 'pt_BR',
    status: 'processing',
    total_recipients: 100,
    sent_count: 40,
    delivered_count: 30,
    read_count: 10,
    failed_count: 10,
    created_at: '2026-06-01T00:00:00Z',
    updated_at: '2026-06-01T00:00:00Z',
  },
  {
    id: 'camp-2',
    tenant_id: 'tenant-1',
    channel_id: 'ch-1',
    name: 'Welcome Wave',
    template_name: 'welcome_v1',
    template_language: 'pt_BR',
    status: 'draft',
    total_recipients: 50,
    sent_count: 0,
    delivered_count: 0,
    read_count: 0,
    failed_count: 0,
    created_at: '2026-06-02T00:00:00Z',
    updated_at: '2026-06-02T00:00:00Z',
  },
  {
    id: 'camp-3',
    tenant_id: 'tenant-1',
    channel_id: 'ch-1',
    name: 'March Newsletter',
    template_name: 'newsletter_march',
    template_language: 'en_US',
    status: 'completed',
    total_recipients: 200,
    sent_count: 200,
    delivered_count: 180,
    read_count: 90,
    failed_count: 0,
    created_at: '2026-05-10T00:00:00Z',
    updated_at: '2026-05-10T00:00:00Z',
  },
  {
    id: 'camp-4',
    tenant_id: 'tenant-1',
    channel_id: 'ch-1',
    name: 'Bad Batch',
    template_name: 'flash_sale',
    template_language: 'pt_BR',
    status: 'failed',
    total_recipients: 10,
    sent_count: 2,
    delivered_count: 1,
    read_count: 0,
    failed_count: 6,
    created_at: '2026-06-03T00:00:00Z',
    updated_at: '2026-06-03T00:00:00Z',
  },
]

const RECIPIENTS = [
  {
    id: 'rec-1',
    campaign_id: 'camp-1',
    phone: '+5511999990001',
    params: ['John'],
    status: 'delivered',
    attempts: 1,
    created_at: '2026-06-01T00:00:00Z',
  },
  {
    id: 'rec-2',
    campaign_id: 'camp-1',
    phone: '+5511999990002',
    params: ['Mary'],
    status: 'failed',
    error_reason: 'invalid number',
    attempts: 3,
    created_at: '2026-06-01T00:00:00Z',
  },
  {
    id: 'rec-3',
    campaign_id: 'camp-1',
    phone: '+5511999990003',
    params: [],
    status: 'pending',
    attempts: 0,
    created_at: '2026-06-01T00:00:00Z',
  },
]

function envelope(data: unknown, meta?: JsonRecord) {
  return JSON.stringify({
    success: true,
    data,
    ...(meta ? { meta } : {}),
  })
}

const RECIPIENTS_META = {
  page: 1,
  page_size: 50,
  total_pages: 1,
  total_items: RECIPIENTS.length,
  has_next: false,
  has_previous: false,
}

async function mockCampaignsEndpoints(page: import('@playwright/test').Page) {
  // Channels for the new-campaign form (only supported outbound types are listed).
  await page.route('**/api/v1/channels**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: envelope([
        {
          id: 'ch-1',
          tenant_id: 'tenant-1',
          name: 'WhatsApp Business',
          type: 'whatsapp_official',
          enabled: true,
          connection_status: 'connected',
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
        },
        {
          id: 'ch-2',
          tenant_id: 'tenant-1',
          name: 'Telegram Bot',
          type: 'telegram',
          enabled: true,
          connection_status: 'connected',
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
        },
      ]),
    })
  })

  // Campaign list (the page polls every 5s, the handler stays registered and
  // answers every refetch). POST is answered generically; the create test
  // registers a more specific capturing route on top of this one.
  await page.route('**/api/v1/campaigns', async (route) => {
    if (route.request().method() === 'POST') {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: envelope({
          ...CAMPAIGNS[1],
          id: 'camp-new',
          name: 'June Blast',
          status: 'draft',
        }),
      })
      return
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: envelope(CAMPAIGNS, {
        page: 1,
        page_size: 20,
        total_items: CAMPAIGNS.length,
      }),
    })
  })

  // Campaign details + recipients (also poll while status === 'processing').
  for (const campaign of CAMPAIGNS) {
    await page.route(`**/api/v1/campaigns/${campaign.id}`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: envelope(campaign),
      })
    })

    await page.route(`**/api/v1/campaigns/${campaign.id}/recipients**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: envelope(campaign.id === 'camp-1' ? RECIPIENTS : [], RECIPIENTS_META),
      })
    })
  }

  // Detail page the create flow redirects to.
  await page.route('**/api/v1/campaigns/camp-new', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: envelope({
        ...CAMPAIGNS[1],
        id: 'camp-new',
        name: 'June Blast',
        total_recipients: 2,
      }),
    })
  })

  await page.route('**/api/v1/campaigns/camp-new/recipients**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: envelope([], { ...RECIPIENTS_META, total_items: 0 }),
    })
  })
}

test.describe('Campaigns', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockEmptyEndpoints(page)
    await mockCampaignsEndpoints(page)
  })

  test('lists campaigns with status badges', async ({ page }) => {
    await page.goto('/campaigns')

    await expect(page.getByText('April Promo')).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('Welcome Wave')).toBeVisible()
    await expect(page.getByText('March Newsletter')).toBeVisible()
    await expect(page.getByText('Bad Batch')).toBeVisible()

    await expect(page.getByText('processing', { exact: true }).first()).toBeVisible()
    await expect(page.getByText('draft', { exact: true }).first()).toBeVisible()
    await expect(page.getByText('completed', { exact: true }).first()).toBeVisible()
    await expect(page.getByText('failed', { exact: true }).first()).toBeVisible()
  })

  test('shows campaign progress', async ({ page }) => {
    await page.goto('/campaigns')

    // April Promo: (40 sent + 10 failed) / 100 = 50%
    await expect(page.getByText('50%')).toBeVisible({ timeout: 15000 })
    // March Newsletter: 200/200 = 100%
    await expect(page.getByText('100%').first()).toBeVisible()
    // Bad Batch: (2 + 6) / 10 = 80%
    await expect(page.getByText('80%')).toBeVisible()
  })

  test('shows empty state when there are no campaigns', async ({ page }) => {
    await page.route('**/api/v1/campaigns', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: envelope([], { page: 1, page_size: 20, total_items: 0 }),
      })
    })

    await page.goto('/campaigns')

    await expect(page.getByText('No campaigns yet')).toBeVisible({ timeout: 15000 })
  })

  test('navigates to the new campaign form', async ({ page }) => {
    await page.goto('/campaigns')

    await expect(page.getByText('April Promo')).toBeVisible({ timeout: 15000 })
    await page.getByRole('button', { name: 'New campaign' }).click()

    await expect(page).toHaveURL(/\/campaigns\/new$/)
    await expect(page.getByRole('heading', { name: 'New campaign' })).toBeVisible()
  })

  test('creates a campaign and asserts the POST payload', async ({ page }) => {
    let createPayload: JsonRecord | null = null

    await page.route('**/api/v1/campaigns', async (route) => {
      if (route.request().method() === 'POST') {
        createPayload = route.request().postDataJSON() as JsonRecord
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: envelope({
            ...CAMPAIGNS[1],
            id: 'camp-new',
            name: 'June Blast',
            total_recipients: 2,
          }),
        })
        return
      }

      await route.fallback()
    })

    await page.goto('/campaigns/new')

    await expect(page.getByRole('heading', { name: 'New campaign' })).toBeVisible({
      timeout: 15000,
    })

    await page.getByLabel('Name').fill('June Blast')

    // shadcn Select: open the trigger, pick a role=option entry.
    await page.getByRole('combobox').click()
    await page
      .getByRole('option', { name: 'WhatsApp Business (whatsapp_official)' })
      .click()

    await page.getByLabel('Template').fill('welcome_v1')
    await page.getByLabel('Language').fill('pt_BR')
    await page
      .getByLabel('Recipients')
      .fill('+5511999990001,John,Order #42\n+5511999990002,Mary')

    await page.getByRole('button', { name: 'Create campaign' }).click()

    await expect.poll(() => createPayload).not.toBeNull()
    expect(createPayload).toMatchObject({
      name: 'June Blast',
      channel_id: 'ch-1',
      template_name: 'welcome_v1',
      template_language: 'pt_BR',
      recipients: [
        { phone: '+5511999990001', params: ['John', 'Order #42'] },
        { phone: '+5511999990002', params: ['Mary'] },
      ],
    })

    // Redirects to the new campaign's detail page.
    await expect(page).toHaveURL(/\/campaigns\/camp-new$/)
    await expect(page.getByRole('heading', { name: 'June Blast' })).toBeVisible()
  })

  test('shows campaign detail with stats and recipients', async ({ page }) => {
    await page.goto('/campaigns/camp-1')

    await expect(page.getByRole('heading', { name: 'April Promo' })).toBeVisible({
      timeout: 15000,
    })

    await expect(page.getByText('processing', { exact: true }).first()).toBeVisible()
    await expect(page.getByText('promo_april · pt_BR')).toBeVisible()
    await expect(page.getByText('50%')).toBeVisible()

    // Recipient rows
    await expect(page.getByText('+5511999990001')).toBeVisible()
    await expect(page.getByText('+5511999990002')).toBeVisible()
    await expect(page.getByText('delivered', { exact: true }).first()).toBeVisible()
    await expect(page.getByText('invalid number')).toBeVisible()
  })

  test('starts a draft campaign', async ({ page }) => {
    let startCalled = false

    await page.route('**/api/v1/campaigns/camp-2/start', async (route) => {
      startCalled = true
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: envelope({}),
      })
    })

    await page.goto('/campaigns/camp-2')

    await expect(page.getByRole('heading', { name: 'Welcome Wave' })).toBeVisible({
      timeout: 15000,
    })

    await page.getByRole('button', { name: 'Start' }).click()

    await expect.poll(() => startCalled).toBe(true)
  })

  test('cancels a processing campaign after confirmation', async ({ page }) => {
    let cancelCalled = false

    await page.route('**/api/v1/campaigns/camp-1/cancel', async (route) => {
      cancelCalled = true
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: envelope({}),
      })
    })

    await page.goto('/campaigns/camp-1')

    await expect(page.getByRole('heading', { name: 'April Promo' })).toBeVisible({
      timeout: 15000,
    })

    await page.getByRole('button', { name: 'Cancel', exact: true }).click()

    const dialog = page.getByRole('alertdialog')
    await expect(dialog.getByText('Cancel this campaign?')).toBeVisible()
    await dialog.getByRole('button', { name: 'Yes, cancel campaign' }).click()

    await expect.poll(() => cancelCalled).toBe(true)
  })
})
