import { test, expect, type Page } from '@playwright/test'
import { setupAuth } from './helpers'

/**
 * A lista de conversas: paginada por cursor, buscável e com a prévia da última
 * mensagem.
 *
 * Os três vinham de uma caixa real: 1.012 conversas, das quais a tela mostrava
 * 20 — sem rolagem, sem paginação, e com `limit` ignorado pela API. A busca
 * mandava o termo e a resposta voltava inteira, e toda linha dizia "ainda sem
 * mensagens" porque a listagem não trazia a última.
 */

const TOTAL_CONVERSAS = 75
const TAMANHO_PAGINA = 30

type ConversaMock = {
  id: string
  tenant_id: string
  channel_id: string
  contact_id: string
  status: string
  priority: string
  unread_count: number
  environment: string
  created_at: string
  updated_at: string
  last_message_at: string
  contact: { id: string; name: string; phone: string }
  channel: { id: string; type: string; name: string }
  last_message: {
    id: string
    content: string
    content_type: string
    sender_type: string
    created_at: string
  }
}

function construirCaixa(): ConversaMock[] {
  const base = Date.parse('2026-10-12T09:00:00Z')
  // Da mais ativa para a mais parada, como o backend devolve.
  return Array.from({ length: TOTAL_CONVERSAS }, (_, i) => {
    const quando = new Date(base - i * 60_000).toISOString()
    return {
      id: `conv-${String(i).padStart(3, '0')}`,
      tenant_id: 'tenant-1',
      channel_id: 'ch-1',
      contact_id: `contact-${i}`,
      status: 'open',
      priority: 'normal',
      unread_count: 0,
      environment: 'production',
      created_at: quando,
      updated_at: quando,
      last_message_at: quando,
      contact: { id: `contact-${i}`, name: `Contato ${i}`, phone: `+55419999${String(i).padStart(4, '0')}` },
      channel: { id: 'ch-1', type: 'whatsapp', name: 'Vendedor 1' },
      last_message: {
        id: `msg-${i}`,
        content: `última fala da conversa ${i}`,
        content_type: 'text',
        sender_type: i % 2 === 0 ? 'contact' : 'user',
        created_at: quando,
      },
    }
  })
}

function cursorDe(conversa: ConversaMock): string {
  return Buffer.from(`${conversa.last_message_at}|${conversa.id}`)
    .toString('base64')
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')
}

/**
 * Serve a caixa como o backend: página por cursor, busca pelo nome do contato,
 * e o total só na primeira página. Registra o que foi pedido para o teste poder
 * afirmar que a tela paginou e buscou de verdade.
 */
async function mockCaixaPaginada(
  page: Page,
  caixa: ConversaMock[],
  pedidos: { before: string; search: string }[]
) {
  await page.route('**/api/v1/conversations**', async (route) => {
    const url = new URL(route.request().url())

    if (url.pathname.endsWith('/messages') || /\/conversations\/[^/]+$/.test(url.pathname)) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] }),
      })
      return
    }

    const before = url.searchParams.get('before') ?? ''
    const search = url.searchParams.get('search') ?? ''
    pedidos.push({ before, search })

    const limite = Number(url.searchParams.get('limit') ?? TAMANHO_PAGINA)
    const filtradas = search
      ? caixa.filter((c) => c.contact.name.toLowerCase().includes(search.toLowerCase()))
      : caixa

    const inicio = before ? filtradas.findIndex((c) => cursorDe(c) === before) + 1 : 0
    const pagina = filtradas.slice(inicio, inicio + limite)
    const temMais = inicio + limite < filtradas.length

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: pagina,
        meta: {
          page_size: pagina.length,
          total_items: before ? 0 : filtradas.length,
          has_next: temMais,
          has_previous: before !== '',
          ...(temMais ? { next_cursor: cursorDe(pagina[pagina.length - 1]) } : {}),
        },
      }),
    })
  })

  await page.route('**/api/v1/channels**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: [{ id: 'ch-1', type: 'whatsapp', name: 'Vendedor 1' }] }),
    })
  )
}

const listaViewport = (page: Page) => page.locator('[data-radix-scroll-area-viewport]').first()

test.describe('Lista de conversas', () => {
  test('rola para o fim e traz as próximas, até a caixa acabar', async ({ page }) => {
    const pedidos: { before: string; search: string }[] = []
    await setupAuth(page, 'admin')
    await mockCaixaPaginada(page, construirCaixa(), pedidos)

    await page.goto('/conversations')
    await expect(page.getByText('Contato 0', { exact: true })).toBeVisible()

    // Abriu com uma página só, e diz quantas existem ao todo.
    await expect(page.getByText(`Contato ${TAMANHO_PAGINA}`, { exact: true })).toHaveCount(0)
    await expect(page.getByText(`Showing ${TAMANHO_PAGINA} of ${TOTAL_CONVERSAS}`)).toBeVisible()
    expect(pedidos).toHaveLength(1)

    await listaViewport(page).evaluate((el) => {
      el.scrollTop = el.scrollHeight
    })
    await expect(page.getByText(`Contato ${TAMANHO_PAGINA}`, { exact: true })).toBeVisible()
    expect(pedidos[1].before).not.toBe('')

    await listaViewport(page).evaluate((el) => {
      el.scrollTop = el.scrollHeight
    })
    await expect(page.getByText(`Contato ${TOTAL_CONVERSAS - 1}`, { exact: true })).toBeVisible()

    // Chegou ao fim: a contagem deixa de ser parcial.
    await expect(page.getByText(`${TOTAL_CONVERSAS} conversations`)).toBeVisible()
  })

  test('a busca vai ao servidor e estreita a lista', async ({ page }) => {
    const pedidos: { before: string; search: string }[] = []
    await setupAuth(page, 'admin')
    await mockCaixaPaginada(page, construirCaixa(), pedidos)

    await page.goto('/conversations')
    await expect(page.getByText('Contato 0', { exact: true })).toBeVisible()

    await page.getByPlaceholder(/search conversations/i).fill('Contato 42')

    await expect(page.getByText('Contato 42', { exact: true })).toBeVisible()
    await expect(page.getByText('Contato 0', { exact: true })).toHaveCount(0)

    const comBusca = pedidos.filter((p) => p.search !== '')
    expect(comBusca.length).toBeGreaterThan(0)
    expect(comBusca[comBusca.length - 1].search).toBe('Contato 42')
    // Uma consulta por tecla faria dez pedidos para "Contato 42".
    expect(comBusca.length).toBeLessThan(4)
  })

  test('mídia sem legenda é anunciada pelo tipo, não como caixa vazia', async ({ page }) => {
    const caixa = construirCaixa()
    // Áudio e documento chegam sem texto — é a natureza deles.
    caixa[0].last_message = { ...caixa[0].last_message, content: '', content_type: 'audio' }
    caixa[1].last_message = { ...caixa[1].last_message, content: '', content_type: 'document' }

    await setupAuth(page, 'admin')
    await mockCaixaPaginada(page, caixa, [])

    await page.goto('/conversations')

    await expect(page.getByText('🎤 Audio')).toBeVisible()
    await expect(page.getByText('📄 Document')).toBeVisible()
    await expect(page.getByText('No messages yet')).toHaveCount(0)
  })

  test('cada linha mostra a última mensagem, não "ainda sem mensagens"', async ({ page }) => {
    await setupAuth(page, 'admin')
    await mockCaixaPaginada(page, construirCaixa(), [])

    await page.goto('/conversations')

    await expect(page.getByText('última fala da conversa 0')).toBeVisible()
    await expect(page.getByText('No messages yet')).toHaveCount(0)
  })
})
