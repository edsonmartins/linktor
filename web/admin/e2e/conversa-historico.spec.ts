import { test, expect, type Page } from '@playwright/test'
import { setupAuth } from './helpers'

/**
 * A tela de mensagens da conversa: histórico paginado por cursor, alinhamento
 * por lado da conversa e mensagens sem conteúdo exibível.
 *
 * Os três vinham de uma tela real: a conversa inteira descia de uma vez, tudo
 * aparecia à esquerda (as mensagens importadas do aparelho pareado chegam sem
 * sender_id, então nenhuma batia com o usuário logado) e os tipos que o
 * conector não traduz saíam como retângulos cinzas vazios.
 */

const TOTAL_MENSAGENS = 75
const TAMANHO_PAGINA = 30

const conversa = {
  id: 'conv-1',
  channel_id: 'ch-1',
  contact_id: 'contact-1',
  environment: 'production',
  status: 'open',
  priority: 'normal',
  unread_count: 0,
  created_at: '2026-01-01T10:00:00Z',
  updated_at: '2026-01-02T10:00:00Z',
  last_message_at: '2026-01-02T10:00:00Z',
  contact: { id: 'contact-1', name: 'Marcia Ribeiro', phone: '+5541999999999' },
  channel: { id: 'ch-1', type: 'whatsapp', environment: 'production' },
}

type MensagemMock = {
  id: string
  conversation_id: string
  sender_type: string
  sender_id: string | null
  content_type: string
  content: string
  status: string
  metadata: Record<string, string>
  attachments: unknown[]
  created_at: string
}

/**
 * Thread sintética, da mais antiga para a mais nova. O remetente alterna e
 * nenhuma mensagem do negócio traz sender_id — é exatamente o dado que chega
 * do aparelho pareado, e o que derrubava o alinhamento.
 */
function construirThread(): MensagemMock[] {
  const base = Date.parse('2026-01-01T10:00:00Z')
  return Array.from({ length: TOTAL_MENSAGENS }, (_, i) => ({
    id: `msg-${String(i).padStart(3, '0')}`,
    conversation_id: 'conv-1',
    sender_type: i % 2 === 0 ? 'contact' : 'user',
    sender_id: null,
    content_type: 'text',
    content: `mensagem ${i}`,
    status: 'delivered',
    metadata: {},
    attachments: [],
    created_at: new Date(base + i * 60_000).toISOString(),
  }))
}

/** Cursor opaco no mesmo formato do backend: base64url de "created_at|id". */
function cursorDe(mensagem: MensagemMock): string {
  return Buffer.from(`${mensagem.created_at}|${mensagem.id}`)
    .toString('base64')
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')
}

/**
 * Serve a thread como o backend: página do mais novo para o mais antigo,
 * começando logo antes do cursor. Registra cada `before` recebido para o teste
 * poder afirmar que a tela paginou de verdade, em vez de ter recebido tudo.
 */
async function mockConversaPaginada(page: Page, thread: MensagemMock[], cursoresPedidos: string[]) {
  await page.route('**/api/v1/conversations**', async (route) => {
    const url = new URL(route.request().url())

    if (url.pathname.endsWith('/messages')) {
      const before = url.searchParams.get('before') ?? ''
      cursoresPedidos.push(before)

      const limite = Number(url.searchParams.get('limit') ?? TAMANHO_PAGINA)
      const doNovoParaOVelho = [...thread].reverse()

      const inicio = before
        ? doNovoParaOVelho.findIndex((m) => cursorDe(m) === before) + 1
        : 0
      const pagina = doNovoParaOVelho.slice(inicio, inicio + limite)
      const temMais = inicio + limite < doNovoParaOVelho.length

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: pagina,
          meta: {
            page_size: pagina.length,
            has_next: temMais,
            has_previous: before !== '',
            ...(temMais ? { next_cursor: cursorDe(pagina[pagina.length - 1]) } : {}),
          },
        }),
      })
      return
    }

    if (/\/conversations\/conv-1(\?|$)/.test(url.href)) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: conversa }),
      })
      return
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: [conversa] }),
    })
  })

  await page.route('**/api/v1/conversations/conv-1/escalation-context', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: {} }),
    })
  )
  await page.route('**/api/v1/users**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: [] }),
    })
  )
}

async function abrirConversa(page: Page) {
  await page.goto('/conversations')
  await page.getByRole('button').filter({ hasText: 'Marcia Ribeiro' }).first().click()
  await expect(page.getByText(`mensagem ${TOTAL_MENSAGENS - 1}`)).toBeVisible()
}

const viewport = (page: Page) => page.locator('[data-radix-scroll-area-viewport]').last()

test.describe('Histórico da conversa', () => {
  test('abre na última página e busca as anteriores ao rolar para cima', async ({ page }) => {
    const cursoresPedidos: string[] = []
    await setupAuth(page, 'admin')
    await mockConversaPaginada(page, construirThread(), cursoresPedidos)
    await abrirConversa(page)

    // Abriu só com a última página: a mais antiga da thread não está na tela.
    await expect(page.getByText('mensagem 0', { exact: true })).toHaveCount(0)
    expect(cursoresPedidos).toEqual([''])

    // A leitura começa no fim, como quem abre um chat.
    const posicaoInicial = await viewport(page).evaluate(
      (el) => el.scrollHeight - el.scrollTop - el.clientHeight
    )
    expect(posicaoInicial).toBeLessThan(80)

    // Rolar ao topo traz a página anterior — e com o cursor da resposta.
    await viewport(page).evaluate((el) => {
      el.scrollTop = 0
    })
    await expect(page.getByText(`mensagem ${TOTAL_MENSAGENS - TAMANHO_PAGINA - 1}`)).toBeVisible()
    expect(cursoresPedidos).toHaveLength(2)
    expect(cursoresPedidos[1]).not.toBe('')

    // Rolando de novo, chega-se ao começo da conversa.
    await viewport(page).evaluate((el) => {
      el.scrollTop = 0
    })
    await expect(page.getByText('mensagem 0', { exact: true })).toBeVisible()
    expect(cursoresPedidos).toHaveLength(3)
  })

  test('a leitura não salta quando as mensagens antigas entram acima', async ({ page }) => {
    await setupAuth(page, 'admin')
    await mockConversaPaginada(page, construirThread(), [])
    await abrirConversa(page)

    // No topo da página carregada está a mensagem 45; é ela que o olho está
    // lendo quando o histórico anterior chega.
    const primeiraDaPagina = page.getByText(`mensagem ${TOTAL_MENSAGENS - TAMANHO_PAGINA}`, {
      exact: true,
    })

    await viewport(page).evaluate((el) => {
      el.scrollTop = 0
    })
    await expect(page.getByText(`mensagem ${TOTAL_MENSAGENS - TAMANHO_PAGINA - 1}`)).toBeVisible()

    // Depois de inserir trinta mensagens acima, ela continua no alto da área
    // visível — e não empurrada para fora dela. Sem a correção, a leitura
    // ficaria presa no começo do histórico recém-chegado, trinta mensagens
    // antes de onde estava.
    const areaVisivel = await viewport(page).boundingBox()
    const ondeFicou = await primeiraDaPagina.boundingBox()
    expect(areaVisivel).not.toBeNull()
    expect(ondeFicou).not.toBeNull()
    expect(ondeFicou!.y).toBeGreaterThan(areaVisivel!.y - 24)
    expect(ondeFicou!.y).toBeLessThan(areaVisivel!.y + 120)

    // E a rolagem não está colada no topo: há histórico acima para rolar.
    const scrollTop = await viewport(page).evaluate((el) => el.scrollTop)
    expect(scrollTop).toBeGreaterThan(200)
  })

  test('o que saiu do negócio fica à direita, mesmo sem sender_id', async ({ page }) => {
    await setupAuth(page, 'admin')
    await mockConversaPaginada(page, construirThread(), [])
    await abrirConversa(page)

    // Índice par = contato, ímpar = negócio; nenhuma das duas tem sender_id.
    const doContato = await page.getByText(`mensagem ${TOTAL_MENSAGENS - 3}`).boundingBox()
    const doNegocio = await page.getByText(`mensagem ${TOTAL_MENSAGENS - 2}`).boundingBox()

    expect(doContato).not.toBeNull()
    expect(doNegocio).not.toBeNull()
    expect(doNegocio!.x).toBeGreaterThan(doContato!.x)
  })

  test('mensagem sem conteúdo exibível é anunciada, não deixa a bolha vazia', async ({ page }) => {
    const thread = construirThread()
    // Como chega uma figurinha ou uma atualização de status pelo conector:
    // tipo texto, sem texto e sem anexo.
    thread[TOTAL_MENSAGENS - 1] = {
      ...thread[TOTAL_MENSAGENS - 1],
      content: '',
      attachments: [],
    }

    await setupAuth(page, 'admin')
    await mockConversaPaginada(page, thread, [])

    await page.goto('/conversations')
    await page.getByRole('button').filter({ hasText: 'Marcia Ribeiro' }).first().click()

    await expect(page.getByText('Message with no displayable content')).toBeVisible()
  })
})
