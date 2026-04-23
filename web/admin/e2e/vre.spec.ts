import { expect, test } from '@playwright/test'
import { setupAuth } from './helpers'

async function mockVREEndpoints(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/channels**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: [
          {
            id: 'ch-1',
            tenant_id: 'tenant-1',
            name: 'WhatsApp Business',
            type: 'whatsapp_official',
            enabled: true,
            connection_status: 'connected',
            config: {},
            created_at: '2026-04-21T00:00:00Z',
            updated_at: '2026-04-21T00:00:00Z',
          },
        ],
      }),
    })
  })

  await page.route('**/api/v1/vre/templates', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        templates: ['menu_opcoes', 'card_produto', 'status_pedido'],
      }),
    })
  })

  await page.route('**/api/v1/vre/templates/**/preview', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        image_base64: SAMPLE_IMAGE_BASE64,
        caption: 'Preview caption',
        follow_up_text: 'Preview follow-up',
        width: 800,
        height: 600,
        format: 'jpeg',
        size_bytes: 10240,
        render_time_ms: 12,
        cache_hit: false,
      }),
    })
  })

  await page.route('**/api/v1/vre/config', async (route) => {
    if (route.request().method() === 'PUT') {
      const payload = route.request().postDataJSON() as Record<string, unknown>
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(payload),
      })
      return
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        tenant_id: 'tenant-1',
        name: 'Linktor Demo',
        logo_url: 'https://example.com/logo.png',
        primary_color: '#0F3460',
        secondary_color: '#E94560',
        accent_color: '#16C79A',
        background: '#FFFFFF',
        text_color: '#1A1A2E',
        muted_color: '#8B95A2',
        font_family: "'DM Sans', sans-serif",
        border_radius: '14px',
        icons: {},
      }),
    })
  })

  await page.route('**/api/v1/vre/render', async (route) => {
    const payload = route.request().postDataJSON() as Record<string, unknown>
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        image_base64: SAMPLE_IMAGE_BASE64,
        caption: payload.caption || 'Rendered caption',
        follow_up_text: payload.follow_up_text || 'Rendered follow-up',
        width: 800,
        height: 600,
        format: payload.format || 'jpeg',
        size_bytes: 20480,
        render_time_ms: 18,
        cache_hit: true,
      }),
    })
  })

  await page.route('**/api/v1/vre/render-and-send', async (route) => {
    const payload = route.request().postDataJSON() as Record<string, unknown>
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        image_base64: SAMPLE_IMAGE_BASE64,
        caption: payload.caption || 'Sent caption',
        follow_up_text: payload.follow_up_text || 'Sent follow-up',
        width: 800,
        height: 600,
        format: payload.format || 'jpeg',
        size_bytes: 24576,
        render_time_ms: 24,
        cache_hit: false,
        delivered: true,
      }),
    })
  })

  await page.route('**/api/v1/vre/cache', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        message: 'Cache invalidated',
      }),
    })
  })

  for (const endpoint of ['conversations', 'contacts', 'bots', 'flows']) {
    await page.route(`**/api/v1/${endpoint}**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: [],
          meta: { page: 1, page_size: 20, total_items: 0 },
        }),
      })
    })
  }
}

const SAMPLE_IMAGE_BASE64 =
  '/9j/4AAQSkZJRgABAQAAAQABAAD/2wCEAAkGBxAQEBAQEA8QDw8PDw8QEA8PDw8PDw8QFREWFhURFRUYHSggGBolGxUVITEhJSkrLi4uFx8zODMsNygtLisBCgoKDg0OGhAQGy8lICYtLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLf/AABEIAAEAAgMBIgACEQEDEQH/xAAbAAABBQEBAAAAAAAAAAAAAAAAAQIDBAUGB//EADoQAAEDAgMFBQYEBwAAAAAAAAEAAgMEEQUSITEGE0FRImFxgZEUkaGxwdHwQlJicoKS4fEVM3PC/8QAGQEAAwEBAQAAAAAAAAAAAAAAAAECAwQF/8QAJREAAQQCAQQDAQAAAAAAAAAAAAECERIDITEEQVEiMmETcf/aAAwDAQACEQMRAD8A9cREQBERAEREAREQBERAEREAREQBERAEREH/2Q=='

test.describe('VRE Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockVREEndpoints(page)
  })

  test('loads VRE runtime and previews a template', async ({ page }) => {
    await page.goto('/vre')

    await expect(page.getByRole('heading', { name: 'VRE' })).toBeVisible()
    await expect(page.getByRole('combobox', { name: 'Template' })).toContainText('menu_opcoes')

    await page.getByRole('button', { name: 'Preview', exact: true }).click()

    const previewResult = page.getByTestId('vre-preview-result')
    await expect(previewResult.getByText('Preview caption')).toBeVisible()
    await expect(previewResult.getByText('Size: 800x600')).toBeVisible()
  })

  test('renders and sends a VRE payload', async ({ page }) => {
    let renderPayload: Record<string, unknown> | null = null
    let sendPayload: Record<string, unknown> | null = null

    await page.route('**/api/v1/vre/render', async (route) => {
      renderPayload = route.request().postDataJSON() as Record<string, unknown>
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          image_base64: SAMPLE_IMAGE_BASE64,
          caption: 'Rendered in test',
          follow_up_text: 'Follow-up in test',
          width: 800,
          height: 600,
          format: 'png',
          size_bytes: 20000,
          render_time_ms: 14,
          cache_hit: true,
        }),
      })
    })

    await page.route('**/api/v1/vre/render-and-send', async (route) => {
      sendPayload = route.request().postDataJSON() as Record<string, unknown>
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          image_base64: SAMPLE_IMAGE_BASE64,
          caption: 'Sent in test',
          follow_up_text: 'Sent follow-up',
          width: 800,
          height: 600,
          format: 'jpeg',
          size_bytes: 24000,
          render_time_ms: 22,
          cache_hit: false,
          delivered: true,
        }),
      })
    })

    await page.goto('/vre')

    await page.getByLabel('Caption').fill('Order ready')
    await page.getByLabel('Follow-up text').fill('Collect at pickup desk')
    await page.getByLabel('Send to').fill('+5511999999999')
    await page.getByLabel('Render data (JSON)').fill(JSON.stringify({
      titulo: 'Pedido pronto',
      subtitulo: 'Retire hoje',
      opcoes: [{ label: 'Confirmar retirada', descricao: 'Responder 1', icone: 'pedido' }],
    }))

    await page.getByRole('button', { name: 'Render', exact: true }).click()
    await page.getByRole('button', { name: 'Render and send' }).click()

    await expect.poll(() => renderPayload).not.toBeNull()
    expect(renderPayload).toMatchObject({
      template_id: 'menu_opcoes',
      caption: 'Order ready',
      follow_up_text: 'Collect at pickup desk',
      format: 'jpeg',
      channel: 'whatsapp',
      channel_id: 'ch-1',
    })

    await expect.poll(() => sendPayload).not.toBeNull()
    expect(sendPayload).toMatchObject({
      send_to: '+5511999999999',
      template_id: 'menu_opcoes',
      channel_id: 'ch-1',
    })

    const renderResult = page.getByTestId('vre-render-result')
    await expect(renderResult.getByText('delivered')).toBeVisible()
    await expect(renderResult.getByText('Caption: Sent in test')).toBeVisible()
  })

  test('saves brand config and invalidates cache', async ({ page }) => {
    let savedConfig: Record<string, unknown> | null = null
    let cacheInvalidated = false

    await page.route('**/api/v1/vre/config', async (route) => {
      if (route.request().method() === 'PUT') {
        savedConfig = route.request().postDataJSON() as Record<string, unknown>
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(savedConfig),
        })
        return
      }

      await route.fallback()
    })

    await page.route('**/api/v1/vre/cache', async (route) => {
      cacheInvalidated = true
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ message: 'Cache invalidated' }),
      })
    })

    await page.goto('/vre')

    await page.getByLabel('Brand name').fill('Linktor Labs')
    await page.getByLabel('Primary color').fill('#112233')
    await page.getByRole('button', { name: 'Save brand config' }).click()
    await page.getByRole('button', { name: 'Invalidate cache' }).click()

    await expect.poll(() => savedConfig).not.toBeNull()
    expect(savedConfig).toMatchObject({
      name: 'Linktor Labs',
      primary_color: '#112233',
    })

    await expect.poll(() => cacheInvalidated).toBe(true)
  })
})
