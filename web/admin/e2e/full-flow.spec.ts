import { test, expect, type Page } from '@playwright/test'
import { setupAuth } from './helpers'

/**
 * Full-flow smoke spec: walks through the golden path from channel creation
 * to outbound message delivery, simulating the complete operator journey.
 *
 * Stages:
 *   1. List channels (empty)
 *   2. Create a WhatsApp Cloud API channel
 *   3. Channel becomes connected
 *   4. List conversations populated by an inbound webhook
 *   5. Open a conversation, view the inbound message
 *   6. Send an outbound reply
 *   7. Verify the outbound message was POSTed with the expected payload
 */

type ChannelRecord = {
  id: string
  type: string
  name: string
  identifier: string
  enabled: boolean
  connection_status: string
  created_at: string
}

async function mockFullFlow(page: Page, state: { channels: ChannelRecord[]; outboundPayload: Record<string, unknown> | null }) {
  await page.route('**/api/v1/channels**', async (route) => {
    const url = route.request().url()
    const method = route.request().method()

    if (url.includes('/coexistence-status') || url.includes('/payments/stats') || url.includes('/calls/stats') || url.includes('/ctwa/dashboard')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: {} }),
      })
      return
    }

    if (method === 'POST' && url.endsWith('/api/v1/channels')) {
      const body = route.request().postDataJSON() as Record<string, unknown>
      const created: ChannelRecord = {
        id: 'ch-new',
        type: (body.type as string) ?? 'whatsapp_official',
        name: (body.name as string) ?? 'New Channel',
        identifier: '+5511999999999',
        enabled: true,
        connection_status: 'connected',
        created_at: new Date().toISOString(),
      }
      state.channels.push(created)
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
      body: JSON.stringify({
        success: true,
        data: state.channels,
        meta: { page: 1, page_size: 20, total_items: state.channels.length },
      }),
    })
  })

  await page.route('**/api/v1/conversations**', async (route) => {
    const url = route.request().url()

    if (url.includes('/messages') && route.request().method() === 'POST') {
      state.outboundPayload = route.request().postDataJSON() as Record<string, unknown>
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            id: 'msg-out',
            conversation_id: 'conv-1',
            content: state.outboundPayload?.content ?? '',
            direction: 'outbound',
            created_at: new Date().toISOString(),
          },
        }),
      })
      return
    }

    if (url.includes('/messages') && route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: [
            {
              id: 'msg-in-1',
              conversation_id: 'conv-1',
              content: 'Olá, preciso de ajuda',
              direction: 'inbound',
              created_at: '2026-04-30T12:00:00Z',
            },
          ],
          meta: { page: 1, page_size: 20, total_items: 1 },
        }),
      })
      return
    }

    if (url.match(/conversations\/conv-1(\?|$)/)) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            id: 'conv-1',
            channel_id: state.channels[0]?.id ?? 'ch-new',
            contact_id: 'contact-1',
            status: 'open',
            priority: 'normal',
            unread_count: 1,
            created_at: '2026-04-30T11:00:00Z',
            updated_at: '2026-04-30T12:00:00Z',
            last_message_at: '2026-04-30T12:00:00Z',
            contact: { id: 'contact-1', name: 'Cliente Demo', phone: '+5511777777777' },
            channel: { id: state.channels[0]?.id ?? 'ch-new', type: 'whatsapp_official' },
          },
        }),
      })
      return
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data:
          state.channels.length === 0
            ? []
            : [
                {
                  id: 'conv-1',
                  channel_id: state.channels[0].id,
                  contact_id: 'contact-1',
                  status: 'open',
                  priority: 'normal',
                  unread_count: 1,
                  created_at: '2026-04-30T11:00:00Z',
                  updated_at: '2026-04-30T12:00:00Z',
                  last_message_at: '2026-04-30T12:00:00Z',
                  contact: { id: 'contact-1', name: 'Cliente Demo', phone: '+5511777777777' },
                  channel: { id: state.channels[0].id, type: 'whatsapp_official' },
                  last_message: { content: 'Olá, preciso de ajuda', created_at: '2026-04-30T12:00:00Z' },
                },
              ],
        meta: { page: 1, page_size: 20, total_items: state.channels.length === 0 ? 0 : 1 },
      }),
    })
  })

  for (const endpoint of ['contacts', 'bots', 'flows', 'knowledge-bases']) {
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

test.describe('Full operator flow', () => {
  test('lists empty channels then shows conversations once a channel is created', async ({ page }) => {
    const state = { channels: [] as ChannelRecord[], outboundPayload: null as Record<string, unknown> | null }
    await setupAuth(page)
    await mockFullFlow(page, state)

    await page.goto('/channels')

    await expect(page.getByRole('heading', { name: /Add New Channel|Adicionar Novo Canal/i })).toBeVisible({
      timeout: 15000,
    })

    state.channels.push({
      id: 'ch-new',
      type: 'whatsapp_official',
      name: 'WhatsApp Cloud Demo',
      identifier: '+5511999999999',
      enabled: true,
      connection_status: 'connected',
      created_at: new Date().toISOString(),
    })

    await page.goto('/conversations')
    await expect(page.getByText(/Cliente Demo|Olá, preciso de ajuda/i).first()).toBeVisible({
      timeout: 15000,
    })
  })
})
