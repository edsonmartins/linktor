import { test, expect } from '@playwright/test'
import { setupAuth } from './helpers'

async function mockChannels(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/channels**', async (route) => {
    const url = route.request().url()
    const method = route.request().method()

    if (url.includes('/coexistence-status')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            enabled: true,
            status: 'warning',
            days_since_last_echo: 12,
            days_until_disconnect: 2,
            last_echo_at: '2024-01-10T00:00:00Z',
          },
        }),
      })
      return
    }

    if (url.includes('/payments/stats')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          total_payments: 12,
          successful_payments: 10,
          failed_payments: 1,
          total_amount: 245000,
          refunded_amount: 1500,
          currency: 'BRL',
          success_rate: 0.833,
        }),
      })
      return
    }

    if (url.includes('/calls/stats')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          total_calls: 8,
          inbound_calls: 2,
          outbound_calls: 6,
          completed_calls: 5,
          missed_calls: 1,
          failed_calls: 1,
          total_duration: 720,
          average_duration: 90,
        }),
      })
      return
    }

    if (url.includes('/calls') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          calls: [
            {
              id: 'call-1',
              to: '+5511888888888',
              type: 'voice',
              status: 'completed',
              duration: 180,
              created_at: '2024-01-04T00:00:00Z',
            },
            {
              id: 'call-2',
              to: '+5511777777777',
              type: 'video',
              status: 'ringing',
              duration: 0,
              created_at: '2024-01-05T00:00:00Z',
            },
          ],
          limit: 5,
          offset: 0,
        }),
      })
      return
    }

    if (url.includes('/ctwa/dashboard')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          summary: {
            total_referrals: 24,
            total_conversions: 7,
            conversion_rate: 0.291,
            total_value: 4200.5,
            currency: 'BRL',
            average_value: 600.07,
          },
          top_ads: [
            {
              ad_id: 'ad-1',
              ad_name: 'Promo Abril',
              campaign_name: 'Campanha Retencao',
              referrals: 15,
              conversions: 5,
            },
          ],
        }),
      })
      return
    }

    if (url.includes('/payments') && method === 'POST') {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          payment_id: 'payment-123',
          status: 'pending',
          payment_url: 'https://payments.example/p/payment-123',
        }),
      })
      return
    }

    if (url.includes('/calls') && method === 'POST') {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          call_id: 'call-3',
          status: 'initiated',
          created_at: '2024-01-06T00:00:00Z',
        }),
      })
      return
    }

    if (url.includes('/ctwa/conversions') && method === 'POST') {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'conv-1',
          referral_id: 'ref-123',
          conversion_type: 'purchase',
          status: 'converted',
        }),
      })
      return
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: [
          {
            id: 'ch-1',
            type: 'whatsapp_official',
            name: 'WhatsApp Business',
            identifier: '+5511999999999',
            enabled: true,
            connection_status: 'connected',
            created_at: '2024-01-01T00:00:00Z',
          },
          {
            id: 'ch-2',
            type: 'telegram',
            name: 'Telegram Bot',
            identifier: '@mybot',
            enabled: false,
            connection_status: 'disconnected',
            created_at: '2024-01-02T00:00:00Z',
          },
          {
            id: 'ch-3',
            type: 'whatsapp',
            name: 'WhatsApp Device',
            identifier: '+5511888888888',
            enabled: true,
            connection_status: 'connected',
            created_at: '2024-01-03T00:00:00Z',
          },
          {
            id: 'ch-4',
            type: 'whatsapp',
            name: 'WhatsApp Pending Device',
            identifier: '+5511777777777',
            enabled: true,
            connection_status: 'disconnected',
            created_at: '2024-01-04T00:00:00Z',
          },
        ],
        meta: { page: 1, page_size: 20, total_items: 4 },
      }),
    })
  })

  // Mock other endpoints
  for (const endpoint of ['conversations', 'contacts', 'bots', 'flows']) {
    await page.route(`**/api/v1/${endpoint}**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [], meta: { page: 1, page_size: 20, total_items: 0 } }),
      })
    })
  }
}

async function openAddChannelDialog(
  page: import('@playwright/test').Page,
  channelHeading: RegExp | string
) {
  const addChannelSection = page
    .getByRole('heading', { name: /Add New Channel|Adicionar Novo Canal/i })
    .locator('xpath=../..')
  await addChannelSection.getByRole('heading', { name: channelHeading }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  return dialog
}

async function openConfiguredChannelDialog(
  page: import('@playwright/test').Page,
  channelName: string
) {
  const card = page
    .getByRole('heading', { name: channelName })
    .locator('xpath=ancestor::div[contains(@class,"hover:border-primary/30")]')

  await card.locator('button[aria-haspopup="menu"]').click()
  await page.getByRole('menuitem', { name: /Configure|Configurar/i }).click()

  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  return dialog
}

async function installPopupRedirectMock(
  page: import('@playwright/test').Page,
  options: {
    popupRedirectURL?: string
    mainWindowRedirectURL?: string
  }
) {
  await page.addInitScript((config) => {
    window.open = () => {
      const popup = {
        closed: false,
        location: { href: 'about:blank' },
        close() {
          this.closed = true
        },
      }

      setTimeout(() => {
        if (config.popupRedirectURL) {
          popup.location.href = config.popupRedirectURL
        }
        if (config.mainWindowRedirectURL) {
          window.history.replaceState({}, '', config.mainWindowRedirectURL)
        }
      }, 50)

      setTimeout(() => {
        popup.closed = true
      }, 100)

      return popup as Window
    }
  }, options)
}

test.describe('Channels Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockChannels(page)
  })

  test('lists existing channels', async ({ page }) => {
    await page.goto('/channels')

    await expect(page.locator('h3').filter({ hasText: 'WhatsApp Business' })).toBeVisible({ timeout: 15000 })
    await expect(page.locator('h3').filter({ hasText: 'Telegram Bot' })).toBeVisible()
  })

  test('shows channel type badges', async ({ page }) => {
    await page.goto('/channels')

    await expect(page.getByText(/whatsapp/i).first()).toBeVisible({ timeout: 15000 })
  })

  test('shows channel status', async ({ page }) => {
    await page.goto('/channels')

    await expect(page.locator('h3').filter({ hasText: 'WhatsApp Business' })).toBeVisible({ timeout: 15000 })
    await expect(page.getByText(/connected/i).first()).toBeVisible()
  })

  test('shows coexistence status badge for WhatsApp official channels', async ({ page }) => {
    await page.goto('/channels')

    await expect(page.getByText(/warning/i).first()).toBeVisible()
  })

  test('shows channel operations for WhatsApp official and submits payment, call, and CTWA actions', async ({ page }) => {
    let paymentPayload: Record<string, unknown> | null = null
    let callPayload: Record<string, unknown> | null = null
    let conversionPayload: Record<string, unknown> | null = null

    await page.route('**/api/v1/channels/ch-1/payments', async (route) => {
      paymentPayload = route.request().postDataJSON() as Record<string, unknown>
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          payment_id: 'payment-asserted',
          status: 'pending',
        }),
      })
    })

    await page.route('**/api/v1/channels/ch-1/calls', async (route) => {
      if (route.request().method() === 'POST') {
        callPayload = route.request().postDataJSON() as Record<string, unknown>
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            call_id: 'call-asserted',
            status: 'initiated',
          }),
        })
        return
      }

      await route.fallback()
    })

    await page.route('**/api/v1/channels/ch-1/ctwa/conversions', async (route) => {
      conversionPayload = route.request().postDataJSON() as Record<string, unknown>
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'conversion-asserted',
          status: 'converted',
        }),
      })
    })

    await page.goto('/channels')

    const dialog = await openConfiguredChannelDialog(page, 'WhatsApp Business')
    await dialog.getByRole('tab', { name: 'Operations' }).click()

    await expect(dialog.getByText('Total payments')).toBeVisible()
    await expect(dialog.getByText('R$ 2.450,00')).toBeVisible()
    await expect(dialog.getByText('Recent calls')).toBeVisible()
    await expect(dialog.getByText('Promo Abril')).toBeVisible()

    await dialog.getByLabel('Customer phone').fill('+5511999999999')
    await dialog.getByLabel('Amount (cents)').fill('1999')
    await dialog.getByLabel('Reference ID').fill('order-123')
    await dialog.getByLabel('Description').fill('Order payment')
    await dialog.getByRole('button', { name: 'Create payment' }).click()

    await dialog.getByLabel('Call destination').fill('+5511888888888')
    await dialog.getByLabel('Call type').fill('video')
    await dialog.getByRole('button', { name: 'Start call' }).click()

    await dialog.getByLabel('Referral ID').fill('ref-123')
    await dialog.getByLabel('Conversion type').fill('purchase')
    await dialog.getByLabel('Value').fill('199.9')
    await dialog.getByRole('button', { name: 'Track conversion' }).click()

    await expect.poll(() => paymentPayload).not.toBeNull()
    expect(paymentPayload).toMatchObject({
      to: '+5511999999999',
      reference_id: 'order-123',
      amount: 1999,
      type: 'order',
      currency: 'BRL',
    })

    await expect.poll(() => callPayload).not.toBeNull()
    expect(callPayload).toMatchObject({
      to: '+5511888888888',
      type: 'video',
    })

    await expect.poll(() => conversionPayload).not.toBeNull()
    expect(conversionPayload).toMatchObject({
      referral_id: 'ref-123',
      conversion_type: 'purchase',
      value: 199.9,
      currency: 'BRL',
    })
  })

  test('shows Facebook OAuth connection surface', async ({ page }) => {
    await page.goto('/channels')

    const dialog = await openAddChannelDialog(page, 'Facebook Messenger')
    await dialog.getByRole('tab', { name: /Connect/i }).click()

    await expect(dialog.getByRole('heading', { name: /Connect with Facebook/i })).toBeVisible()
    await expect(dialog.getByText(/^App ID$/i)).toBeVisible()
    await expect(dialog.locator('input').nth(0)).toBeVisible()
    await expect(dialog.getByText(/^App Secret$/i)).toBeVisible()
    await expect(dialog.locator('input').nth(1)).toBeVisible()
    await expect(dialog.getByRole('button', { name: /Connect with Facebook/i })).toBeVisible()
  })

  test('completes Facebook OAuth flow in the frontend', async ({ page }) => {
    await installPopupRedirectMock(page, {
      popupRedirectURL: 'http://localhost:3000/oauth/facebook/callback?code=fb-code&state=fb-state',
    })

    await page.route('**/api/v1/oauth/facebook/login', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            login_url: 'https://facebook.example/oauth',
            state: 'fb-state',
          },
        }),
      })
    })

    await page.route('**/api/v1/oauth/facebook/callback', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            user_access_token: 'user-token',
            pages: [
              {
                id: 'page-1',
                name: 'Support Page',
                access_token: 'page-token',
                category: 'Business',
              },
            ],
          },
        }),
      })
    })

    await page.route('**/api/v1/oauth/channels', async (route) => {
      const body = route.request().postDataJSON()
      expect(body.type).toBe('facebook')
      expect(body.page_id).toBe('page-1')
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            channel: {
              id: 'facebook-oauth-channel',
              type: 'facebook',
              name: 'Support Page',
              enabled: true,
              connection_status: 'connected',
            },
          },
        }),
      })
    })

    await page.goto('/channels')
    const dialog = await openAddChannelDialog(page, 'Facebook Messenger')
    await dialog.getByRole('tab', { name: /Connect/i }).click()
    await dialog.locator('input').nth(0).fill('app-id')
    await dialog.locator('input').nth(1).fill('app-secret')
    await dialog.getByRole('button', { name: /Connect with Facebook/i }).click()

    await expect(dialog.getByText('Support Page')).toBeVisible()
    await dialog.getByText('Support Page').click()
    await dialog.getByRole('button', { name: /Create Channel/i }).first().click()

    await expect
      .poll(async () => await page.evaluate(() => sessionStorage.getItem('fb_oauth_state')))
      .toBeNull()
  })

  test('completes Instagram OAuth flow in the frontend', async ({ page }) => {
    await installPopupRedirectMock(page, {
      popupRedirectURL: 'http://localhost:3000/oauth/instagram/callback?code=ig-code&state=ig-state',
    })

    await page.route('**/api/v1/oauth/instagram/login', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            login_url: 'https://facebook.example/instagram-oauth',
            state: 'ig-state',
          },
        }),
      })
    })

    await page.route('**/api/v1/oauth/instagram/callback', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            user_access_token: 'user-token',
            accounts: [
              {
                id: 'ig-1',
                username: 'supportbrand',
                name: 'Support Brand',
                followers_count: 1200,
                page_id: 'fb-page-1',
                page_name: 'Support Brand Page',
                page_access_token: 'page-token',
              },
            ],
          },
        }),
      })
    })

    await page.route('**/api/v1/oauth/channels', async (route) => {
      const body = route.request().postDataJSON()
      expect(body.type).toBe('instagram')
      expect(body.instagram_id).toBe('ig-1')
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            channel: {
              id: 'instagram-oauth-channel',
              type: 'instagram',
              name: '@supportbrand',
              enabled: true,
              connection_status: 'connected',
            },
          },
        }),
      })
    })

    await page.goto('/channels')

    const dialog = await openAddChannelDialog(page, 'Instagram')
    await dialog.getByRole('tab', { name: /Connect/i }).click()
    await dialog.locator('input').nth(0).fill('app-id')
    await dialog.locator('input').nth(1).fill('app-secret')
    await dialog.getByRole('button', { name: /Connect via Facebook/i }).click()

    await expect(dialog.getByText('@supportbrand')).toBeVisible()
    await dialog.getByText('@supportbrand').click()
    await dialog.getByRole('button', { name: /Create Channel/i }).first().click()

    await expect
      .poll(async () => await page.evaluate(() => sessionStorage.getItem('ig_oauth_state')))
      .toBeNull()
  })

  test('completes WhatsApp Embedded Signup flow in the frontend', async ({ page }) => {
    await installPopupRedirectMock(page, {
      mainWindowRedirectURL: 'http://localhost:3000/channels?code=wa-code&state=wa-state',
    })

    await page.route('**/api/v1/oauth/whatsapp/embedded-signup/start', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            login_url: 'https://facebook.example/embedded-signup',
            state: 'wa-state',
          },
        }),
      })
    })

    await page.route('**/api/v1/oauth/whatsapp/embedded-signup/callback', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            access_token: 'wa-access-token',
            waba_id: 'waba-1',
            phone_number_id: 'phone-id-1',
            phone_number: '+5511999999999',
            business_name: 'WhatsApp Brand',
            verify_token: 'verify-token',
            is_coexistence: true,
            quality_rating: 'GREEN',
          },
        }),
      })
    })

    await page.route('**/api/v1/oauth/whatsapp/embedded-signup/create-channel', async (route) => {
      const body = route.request().postDataJSON()
      expect(body.name).toBe('WhatsApp Brand')
      expect(body.phone_number_id).toBe('phone-id-1')
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            channel: {
              id: 'wa-embedded-channel',
              type: 'whatsapp_official',
              name: 'WhatsApp Brand',
              enabled: true,
              connection_status: 'connected',
            },
          },
        }),
      })
    })

    await page.goto('/channels')

    const dialog = await openAddChannelDialog(page, 'WhatsApp Business')
    await dialog.getByLabel(/Facebook App ID/i).fill('app-id')
    await dialog.getByRole('button', { name: /Connect with Facebook/i }).click()

    await expect(dialog.getByText(/WhatsApp Connected!/i)).toBeVisible()
    await expect(dialog.getByText('+5511999999999')).toBeVisible()
    await expect(dialog.locator('#channelName')).toHaveValue('WhatsApp Brand')
    await dialog.getByRole('button', { name: /Create Channel/i }).first().click()

    await expect
      .poll(async () => await page.evaluate(() => sessionStorage.getItem('wa_embedded_signup_state')))
      .toBeNull()
    await expect(page).toHaveURL(/\/channels$/)
  })

  test('connects a WhatsApp unofficial channel with QR in the frontend', async ({ page }) => {
    await page.route('**/api/v1/channels/ch-4/connect', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            channel: {
              id: 'ch-4',
              type: 'whatsapp',
            },
            qr_code: 'qr-payload',
            expires_in: 60,
          },
        }),
      })
    })

    await page.goto('/channels')

    const card = page.getByRole('heading', { name: 'WhatsApp Pending Device' }).locator('xpath=ancestor::div[contains(@class, "hover:border-primary/30")]')
    await card.locator('button[aria-haspopup="menu"]').click()
    await page.getByRole('menuitem', { name: /Configure/i }).click()

    const dialog = page.getByRole('dialog')
    await dialog.getByRole('tab', { name: /Connection/i }).click()
    await dialog.getByRole('button', { name: /Connect with QR Code/i }).click()

    await expect(dialog.getByAltText('WhatsApp QR Code')).toBeVisible()
    await expect(dialog.getByText(/Scan with WhatsApp/i)).toBeVisible()
  })

  test('requests pair code for a WhatsApp unofficial channel through the pair endpoint', async ({ page }) => {
    await page.route('**/api/v1/channels/ch-4/pair', async (route) => {
      const body = route.request().postDataJSON()
      expect(body.phone_number).toBe('+5511999999999')
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            channel: {
              id: 'ch-4',
              type: 'whatsapp',
            },
            code: '123-456',
            expires_in: 300,
          },
        }),
      })
    })

    await page.goto('/channels')

    const card = page.getByRole('heading', { name: 'WhatsApp Pending Device' }).locator('xpath=ancestor::div[contains(@class, "hover:border-primary/30")]')
    await card.locator('button[aria-haspopup="menu"]').click()
    await page.getByRole('menuitem', { name: /Configure/i }).click()

    const dialog = page.getByRole('dialog')
    await dialog.getByRole('tab', { name: /Connection/i }).click()
    await dialog.locator('input[name="phone_number"]').fill('+5511999999999')
    await dialog.getByRole('button', { name: /Get Pair Code/i }).click()

    await expect(dialog.getByText('123-456')).toBeVisible()
  })

  test('shows connected status and disconnects a WhatsApp unofficial channel', async ({ page }) => {
    await page.route('**/api/v1/channels/ch-3/disconnect', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {},
        }),
      })
    })

    await page.goto('/channels')

    const card = page.getByRole('heading', { name: 'WhatsApp Device' }).locator('xpath=ancestor::div[contains(@class, "hover:border-primary/30")]')
    await card.locator('button[aria-haspopup="menu"]').click()
    await page.getByRole('menuitem', { name: /Configure/i }).click()

    const dialog = page.getByRole('dialog')
    await dialog.getByRole('tab', { name: /Connection/i }).click()

    await expect(dialog.getByText(/Connected/i).first()).toBeVisible()
    await dialog.getByRole('button', { name: /^Disconnect$/i }).click()
    await expect(dialog.getByRole('button', { name: /Connect with QR Code/i })).toBeVisible()
  })

  test('creates a Microsoft Teams channel with required credentials', async ({ page }) => {
    let createPayload: Record<string, unknown> | null = null

    await page.route('**/api/v1/channels', async (route) => {
      if (route.request().method() === 'POST') {
        createPayload = route.request().postDataJSON() as Record<string, unknown>
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              id: 'teams-1',
              type: 'teams',
              name: 'My Teams',
              enabled: true,
              connection_status: 'connected',
            },
          }),
        })
        return
      }
      await route.fallback()
    })

    await page.goto('/channels')

    const dialog = await openAddChannelDialog(page, 'Microsoft Teams')
    await dialog.locator('input[name="name"]').fill('My Teams')
    await dialog.locator('input[name="app_id"]').fill('teams-app-id')
    await dialog.locator('input[name="app_password"]').fill('teams-app-pass')
    await dialog.getByRole('button', { name: /Create Channel/i }).click()

    await expect.poll(() => createPayload).not.toBeNull()
    expect(createPayload).toMatchObject({ type: 'teams' })
    const teamsCreds = (createPayload!.credentials) as Record<string, unknown>
    expect(teamsCreds.app_id).toBe('teams-app-id')
    expect(teamsCreds.app_password).toBe('teams-app-pass')
  })

  test('creates a Slack channel with required credentials', async ({ page }) => {
    let createPayload: Record<string, unknown> | null = null

    await page.route('**/api/v1/channels', async (route) => {
      if (route.request().method() === 'POST') {
        createPayload = route.request().postDataJSON() as Record<string, unknown>
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              id: 'slack-1',
              type: 'slack',
              name: 'My Slack',
              enabled: true,
              connection_status: 'connected',
            },
          }),
        })
        return
      }
      await route.fallback()
    })

    await page.goto('/channels')

    const dialog = await openAddChannelDialog(page, 'Slack')
    await dialog.locator('input[name="name"]').fill('My Slack')
    await dialog.locator('input[name="bot_token"]').fill('xoxb-test-token')
    await dialog.locator('input[name="signing_secret"]').fill('signing-secret-value')
    await dialog.locator('input[name="webhook_url"]').fill('https://desklenz.example.com/webhooks/linktor')
    await dialog.locator('input[name="webhook_secret"]').fill('outbound-hmac-key')
    await dialog.getByRole('button', { name: /Create Channel/i }).click()

    await expect.poll(() => createPayload).not.toBeNull()
    expect(createPayload).toMatchObject({
      type: 'slack',
      webhook_url: 'https://desklenz.example.com/webhooks/linktor',
    })
    const slackCreds = (createPayload!.credentials) as Record<string, unknown>
    expect(slackCreds.bot_token).toBe('xoxb-test-token')
    expect(slackCreds.signing_secret).toBe('signing-secret-value')
    expect(slackCreds.webhook_secret).toBe('outbound-hmac-key')
  })

  test('creates a Mattermost channel with required credentials', async ({ page }) => {
    let createPayload: Record<string, unknown> | null = null

    await page.route('**/api/v1/channels', async (route) => {
      if (route.request().method() === 'POST') {
        createPayload = route.request().postDataJSON() as Record<string, unknown>
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              id: 'mattermost-1',
              type: 'mattermost',
              name: 'My Mattermost',
              enabled: true,
              connection_status: 'connected',
            },
          }),
        })
        return
      }
      await route.fallback()
    })

    await page.goto('/channels')

    const dialog = await openAddChannelDialog(page, 'Mattermost')
    await dialog.locator('input[name="name"]').fill('My Mattermost')
    await dialog.locator('input[name="base_url"]').fill('https://mattermost.example.com')
    await dialog.locator('input[name="bot_token"]').fill('mm-bot-token')
    await dialog.getByRole('button', { name: /Create Channel/i }).click()

    await expect.poll(() => createPayload).not.toBeNull()
    expect(createPayload).toMatchObject({ type: 'mattermost' })
    const mmCreds = (createPayload!.credentials) as Record<string, unknown>
    expect(mmCreds.base_url).toBe('https://mattermost.example.com')
    expect(mmCreds.bot_token).toBe('mm-bot-token')
  })
})
