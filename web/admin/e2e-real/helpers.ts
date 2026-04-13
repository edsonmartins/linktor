import { expect, type Page, type APIRequestContext } from '@playwright/test'

export async function assertApiHealthy(request: APIRequestContext) {
  const health = await request.get('http://localhost:8081/health')
  expect(health.ok()).toBeTruthy()
}

export async function loginAsAdminApi(request: APIRequestContext) {
  const response = await request.post('http://localhost:8081/api/v1/auth/login', {
    data: {
      email: 'admin@demo.com',
      password: 'admin123',
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  const accessToken = payload?.data?.access_token as string
  expect(accessToken).toBeTruthy()

  return accessToken
}

export async function listUsers(request: APIRequestContext) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.get('http://localhost:8081/api/v1/users', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return (payload?.data || []) as Array<{ id: string; email: string; name: string }>
}

export async function createUserByApi(
  request: APIRequestContext,
  input: { name: string; email: string; password: string; role?: 'admin' | 'supervisor' | 'agent' }
) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.post('http://localhost:8081/api/v1/users', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
    data: {
      name: input.name,
      email: input.email,
      password: input.password,
      role: input.role || 'agent',
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return payload?.data as { id: string; email: string; name: string }
}

export async function deleteUserByEmail(request: APIRequestContext, email: string) {
  const accessToken = await loginAsAdminApi(request)
  const usersResponse = await request.get('http://localhost:8081/api/v1/users', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(usersResponse.ok()).toBeTruthy()
  const payload = await usersResponse.json()
  const users = (payload?.data || []) as Array<{ id: string; email: string }>
  const user = users.find((item) => item.email === email)
  if (!user) return

  const deleteResponse = await request.delete(`http://localhost:8081/api/v1/users/${user.id}`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(deleteResponse.ok()).toBeTruthy()
}

export async function findUserByEmail(request: APIRequestContext, email: string) {
  const users = await listUsers(request)
  return users.find((user) => user.email === email) || null
}

export async function listChannels(request: APIRequestContext) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.get('http://localhost:8081/api/v1/channels', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return (payload?.data || []) as Array<{
    id: string
    name: string
    type: string
    enabled: boolean
    connection_status?: string
    config?: Record<string, string>
    credentials?: Record<string, string>
    created_at?: string
  }>
}

export async function findChannelByName(request: APIRequestContext, name: string) {
  const channels = await listChannels(request)
  return channels.find((channel) => channel.name === name) || null
}

export async function deleteChannelByName(request: APIRequestContext, name: string) {
  const accessToken = await loginAsAdminApi(request)
  const channelsResponse = await request.get('http://localhost:8081/api/v1/channels', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(channelsResponse.ok()).toBeTruthy()
  const payload = await channelsResponse.json()
  const channels = (payload?.data || []) as Array<{ id: string; name: string }>
  const channel = channels.find((item) => item.name === name)
  if (!channel) return

  const deleteResponse = await request.delete(`http://localhost:8081/api/v1/channels/${channel.id}`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(deleteResponse.ok()).toBeTruthy()
}

export async function deleteChannelByID(request: APIRequestContext, id: string) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.delete(`http://localhost:8081/api/v1/channels/${id}`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
}

export async function createWebchatChannelByApi(
  request: APIRequestContext,
  input: { name: string }
) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.post('http://localhost:8081/api/v1/channels', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
    data: {
      name: input.name,
      type: 'webchat',
      config: {
        primary_color: '#6366f1',
        text_color: '#ffffff',
        position: 'bottom-right',
        welcome_message: 'Hello from Playwright',
        placeholder_text: 'Type a message...',
        auto_open: 'false',
        auto_open_delay: '3',
        show_typing_indicator: 'true',
        allowed_domains: '',
      },
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return payload?.data as { id: string; name: string; type: string }
}

export async function createContactByApi(
  request: APIRequestContext,
  input: { name: string; email: string; phone: string }
) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.post('http://localhost:8081/api/v1/contacts', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
    data: {
      name: input.name,
      email: input.email,
      phone: input.phone,
      tags: ['playwright'],
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return payload?.data as { id: string; name: string; email?: string; phone?: string }
}

export async function listContacts(request: APIRequestContext) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.get('http://localhost:8081/api/v1/contacts', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return (payload?.data || []) as Array<{ id: string; name: string; email?: string; phone?: string; tags?: string[] }>
}

export async function findContactByEmail(request: APIRequestContext, email: string) {
  const contacts = await listContacts(request)
  return contacts.find((contact) => contact.email === email) || null
}

export async function deleteContactByEmail(request: APIRequestContext, email: string) {
  const contact = await findContactByEmail(request, email)
  if (!contact) return

  await deleteContactByID(request, contact.id)
}

export async function addContactIdentityByApi(
  request: APIRequestContext,
  contactID: string,
  input: { channelType: string; identifier: string }
) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.post(`http://localhost:8081/api/v1/contacts/${contactID}/identities`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
    data: {
      channel_type: input.channelType,
      identifier: input.identifier,
      metadata: {
        source: 'playwright',
      },
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return payload?.data as { id: string; identities?: Array<{ id: string; identifier: string; channel_type: string }> }
}

export async function deleteContactByID(request: APIRequestContext, contactID: string) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.delete(`http://localhost:8081/api/v1/contacts/${contactID}`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
}

export async function listKnowledgeBases(request: APIRequestContext) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.get('http://localhost:8081/api/v1/knowledge-bases', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return (payload?.data || []) as Array<{ id: string; name: string; description?: string; type: string; status: string; item_count: number }>
}

export async function createKnowledgeBaseByApi(
  request: APIRequestContext,
  input: { name: string; description?: string; type?: 'faq' | 'documents' | 'website' }
) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.post('http://localhost:8081/api/v1/knowledge-bases', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
    data: {
      name: input.name,
      description: input.description || '',
      type: input.type || 'faq',
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return payload?.data as { id: string; name: string; description?: string; type: string; status: string }
}

export async function findKnowledgeBaseByName(request: APIRequestContext, name: string) {
  const knowledgeBases = await listKnowledgeBases(request)
  return knowledgeBases.find((kb) => kb.name === name) || null
}

export async function deleteKnowledgeBaseByName(request: APIRequestContext, name: string) {
  const accessToken = await loginAsAdminApi(request)
  const knowledgeBase = await findKnowledgeBaseByName(request, name)
  if (!knowledgeBase) return

  const response = await request.delete(`http://localhost:8081/api/v1/knowledge-bases/${knowledgeBase.id}`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
}

export async function listKnowledgeItems(request: APIRequestContext, knowledgeBaseID: string) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.get(`http://localhost:8081/api/v1/knowledge-bases/${knowledgeBaseID}/items`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return (payload?.data || []) as Array<{ id: string; question: string; answer: string; source?: string; keywords?: string[] }>
}

export async function listBots(request: APIRequestContext) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.get('http://localhost:8081/api/v1/bots', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return (payload?.data || []) as Array<{
    id: string
    name: string
    type: string
    provider: string
    model: string
    status: 'active' | 'inactive' | 'training'
    channels?: string[]
  }>
}

export async function createBotByApi(
  request: APIRequestContext,
  input: { name: string; type?: 'customer_service' | 'sales' | 'faq' | 'custom'; provider?: 'openai' | 'anthropic' | 'ollama'; model?: string }
) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.post('http://localhost:8081/api/v1/bots', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
    data: {
      name: input.name,
      type: input.type || 'customer_service',
      provider: input.provider || 'ollama',
      model: input.model || 'llama2',
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return payload?.data as {
    id: string
    name: string
    type: string
    provider: string
    model: string
    status: 'active' | 'inactive' | 'training'
  }
}

export async function findBotByName(request: APIRequestContext, name: string) {
  const bots = await listBots(request)
  return bots.find((bot) => bot.name === name) || null
}

export async function deleteBotByName(request: APIRequestContext, name: string) {
  const accessToken = await loginAsAdminApi(request)
  const bot = await findBotByName(request, name)
  if (!bot) return

  const response = await request.delete(`http://localhost:8081/api/v1/bots/${bot.id}`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
}

export async function getObservabilityLogs(request: APIRequestContext, query = 'limit=1') {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.get(`http://localhost:8081/api/v1/observability/logs?${query}`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
  return await response.json() as { logs: Array<{ id: string; level: string; source: string; message: string }>; total: number; has_more: boolean }
}

export async function getObservabilityQueue(request: APIRequestContext) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.get('http://localhost:8081/api/v1/observability/queue', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
  return await response.json() as { streams: Array<{ name: string }>; total_messages: number; total_pending: number }
}

export async function getObservabilityStats(request: APIRequestContext, period: 'hour' | 'day' | 'week' = 'day') {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.get(`http://localhost:8081/api/v1/observability/stats?period=${period}`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
  return await response.json() as Record<string, unknown>
}

export async function createKnowledgeItemByApi(
  request: APIRequestContext,
  knowledgeBaseID: string,
  input: { question: string; answer: string; source?: string; keywords?: string[] }
) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.post(`http://localhost:8081/api/v1/knowledge-bases/${knowledgeBaseID}/items`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
    data: {
      question: input.question,
      answer: input.answer,
      source: input.source,
      keywords: input.keywords || [],
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return payload?.data as { id: string; question: string; answer: string; source?: string; keywords?: string[] }
}

export async function findKnowledgeItemByQuestion(request: APIRequestContext, knowledgeBaseID: string, question: string) {
  const items = await listKnowledgeItems(request, knowledgeBaseID)
  return items.find((item) => item.question === question) || null
}

export async function getAnalyticsOverview(request: APIRequestContext, period: 'daily' | 'weekly' | 'monthly' = 'weekly') {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.get(`http://localhost:8081/api/v1/analytics/overview?period=${period}`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
  return await response.json() as {
    period: string
    total_messages?: number
    total_conversations?: number
    total_escalations?: number
  }
}

export async function listApiKeys(request: APIRequestContext) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.get('http://localhost:8081/api/v1/api-keys', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return (payload?.data || []) as Array<{ id: string; name: string; key_prefix: string; scopes: string[] }>
}

export async function deleteApiKeyByID(request: APIRequestContext, id: string) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.delete(`http://localhost:8081/api/v1/api-keys/${id}`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
}

export async function createConversationByApi(
  request: APIRequestContext,
  input: { contactID: string; channelID: string; subject?: string }
) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.post('http://localhost:8081/api/v1/conversations', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
    data: {
      contact_id: input.contactID,
      channel_id: input.channelID,
      subject: input.subject || 'Playwright conversation',
      priority: 'medium',
      tags: ['playwright'],
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return payload?.data as { id: string; contact_id: string; channel_id: string; subject?: string }
}

export async function listMessagesByConversation(request: APIRequestContext, conversationID: string) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.get(`http://localhost:8081/api/v1/conversations/${conversationID}/messages`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return (payload?.data || []) as Array<{ id: string; content: string; sender_type: string }>
}

export async function listFlows(request: APIRequestContext) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.get('http://localhost:8081/api/v1/flows', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return (payload?.data || []) as Array<{ id: string; name: string; is_active: boolean; trigger: string; trigger_value?: string }>
}

export async function findFlowByName(request: APIRequestContext, name: string) {
  const flows = await listFlows(request)
  return flows.find((flow) => flow.name === name) || null
}

export async function deleteFlowByName(request: APIRequestContext, name: string) {
  const accessToken = await loginAsAdminApi(request)
  const flow = await findFlowByName(request, name)
  if (!flow) return

  const response = await request.delete(`http://localhost:8081/api/v1/flows/${flow.id}`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
}

export async function loginWithRealApi(page: Page) {
  await page.goto('/login')

  await expect(page.locator('input[type="email"]')).toBeVisible({ timeout: 15000 })
  await page.fill('input[type="email"]', 'admin@demo.com')
  await page.fill('input[type="password"]', 'admin123')
  await page.click('button[type="submit"]')

  await expect(page).toHaveURL(/dashboard/, { timeout: 15000 })
  await assertNoApplicationError(page)
}

export async function assertNoApplicationError(page: Page) {
  await expect(
    page.getByText('Application error: a client-side exception has occurred while loading localhost (see the browser console for more information).')
  ).toHaveCount(0)
}

export async function expectListOrEmptyState(page: Page, options: {
  emptyStateText: RegExp
  primaryContent: () => Promise<boolean>
}) {
  const hasPrimaryContent = await expect
    .poll(async () => await options.primaryContent(), { timeout: 5000 })
    .toBeTruthy()
    .then(() => true)
    .catch(() => false)

  if (hasPrimaryContent) {
    return
  }

  await expect(page.getByText(options.emptyStateText)).toBeVisible()
}

// Settings / Profile helpers

export async function getMe(request: APIRequestContext) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.get('http://localhost:8081/api/v1/me', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return payload?.data as {
    id: string
    name: string
    email: string
    role: string
    avatar_url?: string | null
  }
}

export async function updateMyProfile(
  request: APIRequestContext,
  input: { name?: string; avatar_url?: string }
) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.put('http://localhost:8081/api/v1/me', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
    data: input,
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return payload?.data as { id: string; name: string; email: string }
}

export async function changeMyPassword(
  request: APIRequestContext,
  input: { current_password: string; new_password: string },
  token?: string
) {
  const accessToken = token || (await loginAsAdminApi(request))
  const response = await request.put('http://localhost:8081/api/v1/me/password', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
    data: input,
  })

  expect(response.ok()).toBeTruthy()
}

export async function loginWithPasswordApi(
  request: APIRequestContext,
  email: string,
  password: string
) {
  const response = await request.post('http://localhost:8081/api/v1/auth/login', {
    data: { email, password },
  })
  return { ok: response.ok(), status: response.status() }
}

export async function getTenant(request: APIRequestContext) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.get('http://localhost:8081/api/v1/tenant', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return payload?.data as {
    id: string
    name: string
    slug: string
    plan: string
    settings: Record<string, string>
  }
}

export async function updateTenant(
  request: APIRequestContext,
  input: { name?: string; settings?: Record<string, string> }
) {
  const accessToken = await loginAsAdminApi(request)
  const response = await request.put('http://localhost:8081/api/v1/tenant', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
    data: input,
  })

  expect(response.ok()).toBeTruthy()
  const payload = await response.json()
  return payload?.data as { id: string; name: string; slug: string }
}
