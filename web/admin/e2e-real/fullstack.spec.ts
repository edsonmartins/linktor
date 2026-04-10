import { test, expect } from '@playwright/test'
import {
  addContactIdentityByApi,
  assertApiHealthy,
  assertNoApplicationError,
  createBotByApi,
  createContactByApi,
  createConversationByApi,
  createKnowledgeBaseByApi,
  createKnowledgeItemByApi,
  createWebchatChannelByApi,
  deleteBotByName,
  createUserByApi,
  deleteChannelByID,
  deleteChannelByName,
  deleteContactByEmail,
  deleteContactByID,
  deleteFlowByName,
  deleteKnowledgeBaseByName,
  deleteUserByEmail,
  expectListOrEmptyState,
  findBotByName,
  findKnowledgeBaseByName,
  findKnowledgeItemByQuestion,
  findChannelByName,
  findContactByEmail,
  findFlowByName,
  findUserByEmail,
  getAnalyticsOverview,
  getObservabilityLogs,
  getObservabilityQueue,
  getObservabilityStats,
  listBots,
  listChannels,
  listMessagesByConversation,
  loginAsAdminApi,
  listUsers,
  loginWithRealApi,
} from './helpers'

async function openAddChannelDialog(page: import('@playwright/test').Page, channelHeading: RegExp | string) {
  const addChannelSection = page.getByRole('heading', { name: /Add New Channel|Adicionar Novo Canal/i }).locator('xpath=../..')
  await addChannelSection.getByRole('heading', { name: channelHeading }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  return dialog
}

async function createChannelAndGetNewRecord(
  page: import('@playwright/test').Page,
  request: import('@playwright/test').APIRequestContext,
  options: {
    channelHeading: RegExp | string
    fill: (dialog: import('@playwright/test').Locator, uniqueName: string) => Promise<void>
    expectedType: string
    uniqueName?: boolean
    explicitName?: string
  }
) {
  const before = await listChannels(request)
  const beforeIDs = new Set(before.map((channel) => channel.id))
  const uniqueName = options.explicitName || `Playwright ${options.expectedType} ${Date.now()}`

  const dialog = await openAddChannelDialog(page, options.channelHeading)
  await options.fill(dialog, uniqueName)
  await dialog.locator('button[type="submit"]').click()

  const created = await expect
    .poll(async () => {
      const channels = await listChannels(request)
      const newChannel = channels.find((channel) => !beforeIDs.has(channel.id) && channel.type === options.expectedType)
      return newChannel || null
    }, { timeout: 15000 })
    .toBeTruthy()
    .then(async () => {
      const channels = await listChannels(request)
      return channels.find((channel) => !beforeIDs.has(channel.id) && channel.type === options.expectedType) || null
    })

  expect(created).not.toBeNull()

  if (options.uniqueName !== false) {
    expect(created?.name).toBe(uniqueName)
  }

  return created!
}

test.describe('Admin Full Stack', () => {
  test.beforeEach(async ({ request }) => {
    await assertApiHealthy(request)
  })

  test('logs in against the real API', async ({ page }) => {
    await loginWithRealApi(page)

    await expect(page.getByRole('heading', { name: 'Dashboard', level: 1 })).toBeVisible()
  })

  test('loads dashboard with real backend data', async ({ page }) => {
    await loginWithRealApi(page)

    await page.goto('/dashboard')
    await expect(page.getByRole('heading', { name: 'Dashboard', level: 1 })).toBeVisible()
    await expect(page.getByText('Total Conversations')).toBeVisible()
    await expect(page.getByText('Total Contacts')).toBeVisible()
    await assertNoApplicationError(page)
  })

  test('loads analytics page against the real API', async ({ page, request }) => {
    await getAnalyticsOverview(request, 'weekly')
    await loginWithRealApi(page)

    await page.goto('/analytics')
    await expect(page.getByRole('heading', { name: 'Analytics', level: 1 })).toBeVisible()
    await expect(page.locator('select')).toHaveValue('weekly')
    await assertNoApplicationError(page)
    await expect(page.locator('.rounded-lg.border.bg-card').first()).toBeVisible()
  })

  test('changes analytics period against the real API', async ({ page }) => {
    await loginWithRealApi(page)
    await page.goto('/analytics')

    const overviewResponse = page.waitForResponse((response) =>
      response.url().includes('/api/v1/analytics/overview') &&
      response.url().includes('period=daily') &&
      response.status() === 200
    )

    await page.locator('select').selectOption('daily')
    await overviewResponse

    await expect(page.locator('select')).toHaveValue('daily')
    await assertNoApplicationError(page)
  })

  test('loads observability page against the real API', async ({ page, request }) => {
    await getObservabilityLogs(request)
    await loginWithRealApi(page)

    await page.goto('/observability')
    await expect(page.getByRole('heading', { name: 'Observability', level: 1 })).toBeVisible()
    await expect(page.getByRole('tab', { name: /Logs/i })).toBeVisible()
    await expect(page.getByRole('heading', { name: /Channel Logs/i })).toBeVisible()
    await assertNoApplicationError(page)
  })

  test('switches observability tabs against the real API', async ({ page, request }) => {
    await getObservabilityQueue(request)
    await getObservabilityStats(request, 'day')
    await loginWithRealApi(page)

    await page.goto('/observability')

    const queueResponse = page.waitForResponse((response) =>
      response.url().includes('/api/v1/observability/queue') &&
      response.status() === 200
    )
    await page.getByRole('tab', { name: /Message Queues/i }).click()
    await queueResponse
    await expect(page.getByText(/LINKTOR_MESSAGES|No streams found/i).first()).toBeVisible()

    const statsResponse = page.waitForResponse((response) =>
      response.url().includes('/api/v1/observability/stats') &&
      response.url().includes('period=day') &&
      response.status() === 200
    )
    await page.getByRole('tab', { name: /Statistics/i }).click()
    await statsResponse
    await expect(page.getByRole('tabpanel', { name: /Statistics/i })).toBeVisible()
    await expect(page.getByText(/Connected Channels/i).first()).toBeVisible()
    await assertNoApplicationError(page)
  })

  test('changes observability statistics period against the real API', async ({ page }) => {
    await loginWithRealApi(page)
    await page.goto('/observability')
    await page.getByRole('tab', { name: /Statistics/i }).click()

    const statsResponse = page.waitForResponse((response) =>
      response.url().includes('/api/v1/observability/stats') &&
      response.url().includes('period=week') &&
      response.status() === 200
    )

    await page.getByRole('combobox').click()
    await page.getByRole('option', { name: /Last 7 Days/i }).click()
    await statsResponse
    await expect(page.getByRole('combobox')).toContainText(/Last 7 Days/i)
    await assertNoApplicationError(page)
  })

  test('loads channels page against the real API', async ({ page }) => {
    await loginWithRealApi(page)

    await page.goto('/channels')
    await expect(page.getByRole('heading', { name: 'Channels', level: 1 })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Active Channels' })).toBeVisible()
    await assertNoApplicationError(page)

    await expectListOrEmptyState(page, {
      emptyStateText: /No channels configured|Nenhum canal configurado/i,
      primaryContent: async () => (await page.locator('main h3').count()) > 0,
    })
  })

  test('loads contacts page against the real API', async ({ page }) => {
    await loginWithRealApi(page)

    await page.goto('/contacts')
    await expect(page.getByRole('heading', { name: 'Contacts', level: 1 })).toBeVisible()
    await expect(page.getByPlaceholder(/search contacts/i)).toBeVisible()
    await assertNoApplicationError(page)

    await expectListOrEmptyState(page, {
      emptyStateText: /No contacts found|Nenhum contato encontrado/i,
      primaryContent: async () => (await page.getByRole('main').locator('h3').count()) > 0,
    })
  })

  test('filters contacts by search against the real API', async ({ page, request }) => {
    const suffix = Date.now()
    const targetEmail = `playwright.contact.search.${suffix}@example.com`
    const targetName = `Searchable Contact ${suffix}`
    const otherEmail = `playwright.contact.other.${suffix}@example.com`
    const otherName = `Other Contact ${suffix}`

    await deleteContactByEmail(request, targetEmail)
    await deleteContactByEmail(request, otherEmail)
    await createContactByApi(request, {
      name: targetName,
      email: targetEmail,
      phone: `+1555${String(suffix).slice(-7)}`,
    })
    await createContactByApi(request, {
      name: otherName,
      email: otherEmail,
      phone: `+1666${String(suffix).slice(-7)}`,
    })

    await loginWithRealApi(page)
    await page.goto('/contacts')

    const searchInput = page.getByPlaceholder(/search contacts/i)
    await searchInput.fill(targetEmail)

    await expect(page.getByRole('heading', { name: targetName })).toBeVisible({ timeout: 15000 })
    await expect(page.getByRole('heading', { name: otherName })).toHaveCount(0)

    await deleteContactByEmail(request, targetEmail)
    await deleteContactByEmail(request, otherEmail)
  })

  test('loads conversations page against the real API', async ({ page }) => {
    await loginWithRealApi(page)

    await page.goto('/conversations')
    await expect(page.getByRole('heading', { name: 'Conversations', level: 1 })).toBeVisible()
    await expect(page.getByPlaceholder(/search conversations/i)).toBeVisible()
    await assertNoApplicationError(page)

    await expectListOrEmptyState(page, {
      emptyStateText: /No conversations found|Nenhuma conversa encontrada/i,
      primaryContent: async () => (await page.getByRole('main').locator('button').filter({ hasText: /open|pending|resolved|snoozed/i }).count()) > 0,
    })
  })

  test('loads users page against the real API', async ({ page, request }) => {
    await loginWithRealApi(page)

    const users = await listUsers(request)
    await page.goto('/users')
    await expect(page.getByRole('heading', { name: 'Team', level: 1 })).toBeVisible()
    await expect(page.getByPlaceholder(/search users/i)).toBeVisible()
    await assertNoApplicationError(page)

    if (users.length > 0) {
      await expect(page.getByRole('main').getByRole('heading', { name: users[0].name })).toBeVisible()
    } else {
      await expect(page.getByText(/No team members|Nenhum membro na equipe/i)).toBeVisible()
    }
  })

  test('loads bots page against the real API', async ({ page, request }) => {
    const bots = await listBots(request)
    await loginWithRealApi(page)

    await page.goto('/bots')
    await expect(page.getByRole('heading', { name: 'Bots', level: 1 })).toBeVisible()
    await expect(page.getByPlaceholder(/search bots/i)).toBeVisible()
    await assertNoApplicationError(page)

    if (bots.length > 0) {
      await expect(page.getByRole('main').getByRole('heading', { name: bots[0].name })).toBeVisible({ timeout: 15000 })
    } else {
      await expect(page.getByText(/No bots yet|No Bots/i)).toBeVisible()
    }
  })

  test('creates a bot via UI against the real API', async ({ page, request }) => {
    const botName = `Playwright Bot ${Date.now()}`

    await deleteBotByName(request, botName)
    await loginWithRealApi(page)
    await page.goto('/bots')

    await page.getByRole('button', { name: /create bot/i }).click()
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await dialog.locator('#name').fill(botName)
    await dialog.getByRole('combobox').nth(1).click()
    await page.getByRole('option', { name: /Ollama/i }).click()
    await dialog.getByRole('combobox').nth(2).click()
    await page.getByRole('option', { name: /^llama2$/ }).click()
    await dialog.getByRole('button', { name: /^Create Bot$/ }).click()

    await expect
      .poll(async () => await findBotByName(request, botName), { timeout: 15000 })
      .not.toBeNull()

    await page.reload()
    await expect(page.getByRole('main').getByRole('heading', { name: botName })).toBeVisible({ timeout: 15000 })

    const createdBot = await findBotByName(request, botName)
    expect(createdBot?.name).toBe(botName)

    await deleteBotByName(request, botName)
  })

  test('activates a bot via UI against the real API', async ({ page, request }) => {
    const botName = `Toggle Bot ${Date.now()}`

    await deleteBotByName(request, botName)
    await createBotByApi(request, { name: botName })

    await loginWithRealApi(page)
    await page.goto('/bots')

    const card = page.getByRole('heading', { name: botName }).locator('xpath=ancestor::div[contains(@class, "hover:border-primary/30")]')
    await expect(card.getByRole('heading', { name: botName })).toBeVisible({ timeout: 15000 })

    await card.getByRole('button').click()
    await page.getByRole('menuitem', { name: /Activate/i }).click()

    await expect
      .poll(async () => (await findBotByName(request, botName))?.status, { timeout: 15000 })
      .toBe('active')

    await deleteBotByName(request, botName)
  })

  test('deletes a bot via UI against the real API', async ({ page, request }) => {
    const botName = `Delete Bot ${Date.now()}`

    await deleteBotByName(request, botName)
    await createBotByApi(request, { name: botName })

    await loginWithRealApi(page)
    await page.goto('/bots')

    const card = page.getByRole('heading', { name: botName }).locator('xpath=ancestor::div[contains(@class, "hover:border-primary/30")]')
    await expect(card.getByRole('heading', { name: botName })).toBeVisible({ timeout: 15000 })
    await card.getByRole('button').click()
    await page.getByRole('menuitem', { name: /^Delete$/ }).click()

    const dialog = page.getByRole('alertdialog')
    await expect(dialog).toBeVisible()
    await dialog.getByRole('button', { name: /^Delete$/ }).click()

    await expect
      .poll(async () => await findBotByName(request, botName), { timeout: 15000 })
      .toBeNull()

    await deleteBotByName(request, botName)
  })

  test('loads bot detail page against the real API', async ({ page, request }) => {
    const botName = `Detail Bot ${Date.now()}`

    await deleteBotByName(request, botName)
    const createdBot = await createBotByApi(request, { name: botName })

    await loginWithRealApi(page)
    await page.goto(`/bots/${createdBot.id}`)

    await expect(page.getByText(botName).first()).toBeVisible({ timeout: 15000 })
    await expect(page.getByRole('tab', { name: /General/i })).toBeVisible()
    await expect(page.getByRole('tab', { name: /Channels/i })).toBeVisible()
    await expect(page.getByRole('tab', { name: /Knowledge/i })).toBeVisible()
    await assertNoApplicationError(page)

    await deleteBotByName(request, botName)
  })

  test('creates a user via UI against the real API', async ({ page, request }) => {
    const email = `playwright.user.${Date.now()}@example.com`
    const name = `Playwright User ${Date.now()}`

    await deleteUserByEmail(request, email)
    await loginWithRealApi(page)
    await page.goto('/users')

    await page.getByRole('button', { name: /add member/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible()
    await page.getByLabel('Full Name').fill(name)
    await page.getByLabel('Email').fill(email)
    await page.locator('#password').fill('StrongPass123!')
    await page.locator('#confirmPassword').fill('StrongPass123!')
    await page.getByRole('button', { name: /^Add User$/ }).click()

    await expect(page.getByRole('dialog')).toHaveCount(0, { timeout: 15000 })
    await expect(page.getByRole('main').getByRole('heading', { name })).toBeVisible({ timeout: 15000 })

    await deleteUserByEmail(request, email)
  })

  test('updates a user via UI against the real API', async ({ page, request }) => {
    const email = `playwright.edit.${Date.now()}@example.com`
    const originalName = `Editable User ${Date.now()}`
    const updatedName = `${originalName} Updated`

    await deleteUserByEmail(request, email)
    await createUserByApi(request, {
      name: originalName,
      email,
      password: 'StrongPass123!',
      role: 'agent',
    })

    await loginWithRealApi(page)
    await page.goto('/users')
    await page.getByPlaceholder(/search users/i).fill(email)

    const card = page.getByRole('main').locator('div').filter({
      has: page.getByRole('heading', { name: originalName }),
    }).filter({
      hasText: email,
    }).first()

    await expect(card.getByRole('heading', { name: originalName })).toBeVisible({ timeout: 15000 })
    await card.getByRole('button').last().click()
    await page.getByRole('menuitem', { name: /edit/i }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await dialog.getByLabel('Full Name').fill(updatedName)
    await dialog.getByRole('button', { name: /^Update User$/ }).click()

    await expect(dialog).toHaveCount(0, { timeout: 15000 })
    await expect(page.getByRole('main').getByRole('heading', { name: updatedName })).toBeVisible({ timeout: 15000 })

    const updatedUser = await findUserByEmail(request, email)
    expect(updatedUser?.name).toBe(updatedName)

    await deleteUserByEmail(request, email)
  })

  test('deletes a user via UI against the real API', async ({ page, request }) => {
    const email = `playwright.delete.${Date.now()}@example.com`
    const name = `Disposable User ${Date.now()}`

    await deleteUserByEmail(request, email)
    await createUserByApi(request, {
      name,
      email,
      password: 'StrongPass123!',
      role: 'agent',
    })

    await loginWithRealApi(page)
    await page.goto('/users')
    await page.getByPlaceholder(/search users/i).fill(email)

    const card = page.getByRole('main').locator('div').filter({
      has: page.getByRole('heading', { name }),
    }).filter({
      hasText: email,
    }).first()

    await expect(card.getByRole('heading', { name })).toBeVisible({ timeout: 15000 })
    await card.getByRole('button').last().click()
    await page.getByRole('menuitem', { name: /^Delete$/ }).click()

    const dialog = page.getByRole('alertdialog')
    await expect(dialog).toBeVisible()
    await expect(dialog.getByRole('heading', { name: 'Delete User' })).toBeVisible()
    await dialog.getByRole('button', { name: /^Delete$/ }).click()

    await expect(dialog).toHaveCount(0, { timeout: 15000 })
    await expect(page.getByRole('main').getByRole('heading', { name })).toHaveCount(0)

    const deletedUser = await findUserByEmail(request, email)
    expect(deletedUser).toBeNull()
  })

  test('creates a contact via UI against the real API', async ({ page, request }) => {
    const email = `playwright.contact.${Date.now()}@example.com`
    const name = `Playwright Contact ${Date.now()}`
    const phone = `+1555${String(Date.now()).slice(-7)}`

    await deleteContactByEmail(request, email)
    await loginWithRealApi(page)
    await page.goto('/contacts')

    await page.getByRole('button', { name: /add contact/i }).click()
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await dialog.locator('#contact-name').fill(name)
    await dialog.locator('#contact-email').fill(email)
    await dialog.locator('#contact-phone').fill(phone)
    await dialog.locator('#contact-tags').fill('vip, lead')
    await dialog.getByRole('button', { name: /add contact/i }).click()

    await expect(dialog).toHaveCount(0, { timeout: 15000 })
    await expect(page.getByRole('main').getByRole('heading', { name })).toBeVisible({ timeout: 15000 })

    const createdContact = await findContactByEmail(request, email)
    expect(createdContact?.name).toBe(name)

    await deleteContactByEmail(request, email)
  })

  test('updates a contact via UI against the real API', async ({ page, request }) => {
    const email = `playwright.contact.edit.${Date.now()}@example.com`
    const originalName = `Editable Contact ${Date.now()}`
    const updatedName = `${originalName} Updated`

    await deleteContactByEmail(request, email)
    await createContactByApi(request, {
      name: originalName,
      email,
      phone: `+1555${String(Date.now()).slice(-7)}`,
    })

    await loginWithRealApi(page)
    await page.goto('/contacts')
    await page.getByPlaceholder(/search contacts/i).fill(email)

    const card = page.getByRole('heading', { name: originalName }).locator('xpath=ancestor::div[contains(@class, "hover:border-primary/30")]')

    await expect(card.getByRole('heading', { name: originalName })).toBeVisible({ timeout: 15000 })
    await card.getByRole('button').click()
    await page.getByRole('menuitem', { name: /edit contact/i }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await dialog.locator('#contact-name').fill(updatedName)
    await dialog.locator('button[type="submit"]').click()

    await expect(dialog).toHaveCount(0, { timeout: 15000 })

    await expect
      .poll(async () => {
        const contact = await findContactByEmail(request, email)
        return contact?.name || null
      }, { timeout: 15000 })
      .toBe(updatedName)

    await page.reload()
    await page.getByPlaceholder(/search contacts/i).fill(email)
    await expect(page.getByRole('main').getByRole('heading', { name: updatedName })).toBeVisible({ timeout: 15000 })

    const updatedContact = await findContactByEmail(request, email)
    expect(updatedContact?.name).toBe(updatedName)

    await deleteContactByEmail(request, email)
  })

  test('deletes a contact via UI against the real API', async ({ page, request }) => {
    const email = `playwright.contact.delete.${Date.now()}@example.com`
    const name = `Disposable Contact ${Date.now()}`

    await deleteContactByEmail(request, email)
    await createContactByApi(request, {
      name,
      email,
      phone: `+1555${String(Date.now()).slice(-7)}`,
    })

    await loginWithRealApi(page)
    await page.goto('/contacts')
    await page.getByPlaceholder(/search contacts/i).fill(email)

    const card = page.getByRole('heading', { name }).locator('xpath=ancestor::div[contains(@class, "hover:border-primary/30")]')

    await expect(card.getByRole('heading', { name })).toBeVisible({ timeout: 15000 })
    await card.getByRole('button').click()
    await page.getByRole('menuitem', { name: /^Delete$/ }).click()

    const dialog = page.getByRole('alertdialog')
    await expect(dialog).toBeVisible()
    await expect(dialog.getByRole('heading', { name: /Delete Contact/i })).toBeVisible()
    await dialog.getByRole('button', { name: /^Delete$/ }).click()

    await expect(dialog).toHaveCount(0, { timeout: 15000 })
    await expect
      .poll(async () => await findContactByEmail(request, email), { timeout: 15000 })
      .toBeNull()

    await page.reload()
    await page.getByPlaceholder(/search contacts/i).fill(email)
    await expect(page.getByRole('main').getByRole('heading', { name })).toHaveCount(0)
  })

  test('creates a knowledge base via UI against the real API', async ({ page, request }) => {
    const name = `Playwright Knowledge Base ${Date.now()}`
    const description = 'Knowledge base created by Playwright real E2E'

    await deleteKnowledgeBaseByName(request, name)
    await loginWithRealApi(page)
    await page.goto('/knowledge-base')

    await page.getByRole('button', { name: /new knowledge base/i }).click()
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await dialog.getByLabel('Name').fill(name)
    await dialog.getByLabel('Description').fill(description)
    await dialog.getByRole('button', { name: /^Create$/ }).click()

    await expect(dialog).toHaveCount(0, { timeout: 15000 })
    await expect(page.getByRole('link', { name })).toBeVisible({ timeout: 15000 })

    const createdKnowledgeBase = await findKnowledgeBaseByName(request, name)
    expect(createdKnowledgeBase?.description).toBe(description)

    await deleteKnowledgeBaseByName(request, name)
  })

  test('updates a knowledge base via UI against the real API', async ({ page, request }) => {
    const originalName = `Editable Knowledge Base ${Date.now()}`
    const updatedName = `${originalName} Updated`

    await deleteKnowledgeBaseByName(request, originalName)
    await deleteKnowledgeBaseByName(request, updatedName)
    await createKnowledgeBaseByApi(request, {
      name: originalName,
      description: 'Original KB description',
      type: 'faq',
    })

    await loginWithRealApi(page)
    await page.goto('/knowledge-base')

    const card = page.getByRole('link', { name: originalName }).locator('xpath=ancestor::div[contains(@class, "group transition-colors")]')
    await expect(card.getByRole('link', { name: originalName })).toBeVisible({ timeout: 15000 })
    await card.getByRole('button').click()
    await page.getByRole('menuitem', { name: /^Edit$/ }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await dialog.getByLabel('Name').fill(updatedName)
    await dialog.getByRole('button', { name: /Save Changes/i }).click()

    await expect(dialog).toHaveCount(0, { timeout: 15000 })
    await expect(page.getByRole('link', { name: updatedName })).toBeVisible({ timeout: 15000 })

    const updatedKnowledgeBase = await findKnowledgeBaseByName(request, updatedName)
    expect(updatedKnowledgeBase?.name).toBe(updatedName)

    await deleteKnowledgeBaseByName(request, updatedName)
  })

  test('deletes a knowledge base via UI against the real API', async ({ page, request }) => {
    const name = `Disposable Knowledge Base ${Date.now()}`

    await deleteKnowledgeBaseByName(request, name)
    await createKnowledgeBaseByApi(request, {
      name,
      description: 'Disposable KB',
      type: 'faq',
    })

    await loginWithRealApi(page)
    await page.goto('/knowledge-base')

    const card = page.getByRole('link', { name }).locator('xpath=ancestor::div[contains(@class, "group transition-colors")]')
    await expect(card.getByRole('link', { name })).toBeVisible({ timeout: 15000 })
    await card.getByRole('button').click()
    await page.getByRole('menuitem', { name: /^Delete$/ }).click()

    const dialog = page.getByRole('alertdialog')
    await expect(dialog).toBeVisible()
    await dialog.getByRole('button', { name: /^Delete$/ }).click()

    await expect(dialog).toHaveCount(0, { timeout: 15000 })
    await expect
      .poll(async () => await findKnowledgeBaseByName(request, name), { timeout: 15000 })
      .toBeNull()
  })

  test('creates a knowledge item via UI against the real API', async ({ page, request }) => {
    const kbName = `Knowledge Items ${Date.now()}`
    const question = `How does Playwright item ${Date.now()} work?`
    const answer = 'It is created through the real admin UI.'

    await deleteKnowledgeBaseByName(request, kbName)
    const knowledgeBase = await createKnowledgeBaseByApi(request, {
      name: kbName,
      description: 'KB for item creation',
      type: 'faq',
    })

    await loginWithRealApi(page)
    await page.goto(`/knowledge-base/${knowledgeBase.id}`)

    await page.getByRole('button', { name: /Add Item/i }).click()
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await dialog.getByLabel('Question').fill(question)
    await dialog.getByLabel('Answer').fill(answer)
    await dialog.getByLabel(/Source \(optional\)/i).fill('Playwright E2E')
    await dialog.getByRole('button', { name: /^Add Item$/ }).click()

    await expect(dialog).toHaveCount(0, { timeout: 15000 })
    await expect(page.getByRole('row', { name: new RegExp(question.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')) })).toBeVisible({ timeout: 15000 })

    const createdItem = await findKnowledgeItemByQuestion(request, knowledgeBase.id, question)
    expect(createdItem?.answer).toBe(answer)

    await deleteKnowledgeBaseByName(request, kbName)
  })

  test('updates a knowledge item via UI against the real API', async ({ page, request }) => {
    const kbName = `Editable Items KB ${Date.now()}`
    const originalQuestion = `Original KB Item ${Date.now()}`
    const updatedQuestion = `${originalQuestion} Updated`

    await deleteKnowledgeBaseByName(request, kbName)
    const knowledgeBase = await createKnowledgeBaseByApi(request, {
      name: kbName,
      description: 'KB for item editing',
      type: 'faq',
    })
    await createKnowledgeItemByApi(request, knowledgeBase.id, {
      question: originalQuestion,
      answer: 'Original answer',
      source: 'API seed',
    })

    await loginWithRealApi(page)
    await page.goto(`/knowledge-base/${knowledgeBase.id}`)

    const row = page.getByRole('row').filter({ hasText: originalQuestion }).first()
    await expect(row).toBeVisible({ timeout: 15000 })
    await row.getByRole('button').click()
    await page.getByRole('menuitem', { name: /^Edit$/ }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await dialog.getByLabel('Question').fill(updatedQuestion)
    await dialog.getByRole('button', { name: /Save Changes/i }).click()

    await expect(dialog).toHaveCount(0, { timeout: 15000 })
    await expect(page.getByRole('row').filter({ hasText: updatedQuestion })).toBeVisible({ timeout: 15000 })

    const updatedItem = await findKnowledgeItemByQuestion(request, knowledgeBase.id, updatedQuestion)
    expect(updatedItem?.question).toBe(updatedQuestion)

    await deleteKnowledgeBaseByName(request, kbName)
  })

  test('deletes a knowledge item via UI against the real API', async ({ page, request }) => {
    const kbName = `Disposable Items KB ${Date.now()}`
    const question = `Disposable KB Item ${Date.now()}`

    await deleteKnowledgeBaseByName(request, kbName)
    const knowledgeBase = await createKnowledgeBaseByApi(request, {
      name: kbName,
      description: 'KB for item deletion',
      type: 'faq',
    })
    await createKnowledgeItemByApi(request, knowledgeBase.id, {
      question,
      answer: 'Disposable answer',
      source: 'API seed',
    })

    await loginWithRealApi(page)
    await page.goto(`/knowledge-base/${knowledgeBase.id}`)

    page.once('dialog', async (dialog) => {
      await dialog.accept()
    })

    const row = page.getByRole('row').filter({ hasText: question }).first()
    await expect(row).toBeVisible({ timeout: 15000 })
    await row.getByRole('button').click()
    await page.getByRole('menuitem', { name: /^Delete$/ }).click()

    await expect
      .poll(async () => await findKnowledgeItemByQuestion(request, knowledgeBase.id, question), { timeout: 15000 })
      .toBeNull()

    await deleteKnowledgeBaseByName(request, kbName)
  })

  test('deletes a channel via UI against the real API', async ({ page, request }) => {
    const channelName = `Playwright Delete Channel ${Date.now()}`

    await deleteChannelByName(request, channelName)
    await loginWithRealApi(page)
    await createWebchatChannelByApi(request, { name: channelName })

    await page.goto('/channels')
    const card = page.getByRole('heading', { name: channelName }).locator('xpath=ancestor::div[contains(@class, "hover:border-primary/30")]')

    await expect(card.getByRole('heading', { name: channelName })).toBeVisible({ timeout: 15000 })
    await card.getByRole('button').click()
    await page.getByRole('menuitem', { name: /Delete Channel|Excluir Canal/i }).click()

    const dialog = page.getByRole('alertdialog')
    await expect(dialog).toBeVisible()
    await expect(dialog.getByRole('heading', { name: /Delete Channel|Excluir Canal/i })).toBeVisible()
    await dialog.getByRole('button', { name: /Delete|Excluir/i }).click()

    await expect(dialog).toHaveCount(0, { timeout: 15000 })
    await expect
      .poll(async () => await findChannelByName(request, channelName), { timeout: 15000 })
      .toBeNull()

    await page.reload()
    await expect(page.getByRole('main').getByRole('heading', { name: channelName })).toHaveCount(0)
  })

  test('updates a webchat channel via UI against the real API', async ({ page, request }) => {
    const originalName = `Editable Webchat ${Date.now()}`
    const updatedName = `${originalName} Updated`

    await deleteChannelByName(request, originalName)
    await deleteChannelByName(request, updatedName)
    await createWebchatChannelByApi(request, { name: originalName })

    await loginWithRealApi(page)
    await page.goto('/channels')

    const card = page.getByRole('heading', { name: originalName }).locator('xpath=ancestor::div[contains(@class, "hover:border-primary/30")]')
    await expect(card.getByRole('heading', { name: originalName })).toBeVisible({ timeout: 15000 })
    await card.getByRole('button').click()
    await page.getByRole('menuitem', { name: /Configure/i }).click()

    const sheet = page.getByRole('dialog')
    await expect(sheet).toBeVisible()
    await sheet.locator('#name').fill(updatedName)
    const form = sheet.locator('form')
    await expect(form).toBeVisible()
    const [updateResponse] = await Promise.all([
      page.waitForResponse((response) =>
        response.url().includes('/api/v1/channels/') &&
        !response.url().includes('/enabled') &&
        response.request().method() === 'PUT'
      ),
      form.evaluate((element) => {
        (element as HTMLFormElement).requestSubmit()
      }),
    ])

    expect(updateResponse.status(), await updateResponse.text()).toBe(200)

    await page.reload()
    await expect(page.getByRole('heading', { name: updatedName })).toBeVisible({ timeout: 15000 })

    const updatedChannel = await findChannelByName(request, updatedName)
    expect(updatedChannel?.name).toBe(updatedName)

    await deleteChannelByName(request, updatedName)
  })

  test('toggles a channel enabled state via UI against the real API', async ({ page, request }) => {
    const channelName = `Toggle Channel ${Date.now()}`

    await deleteChannelByName(request, channelName)
    await createWebchatChannelByApi(request, { name: channelName })

    await loginWithRealApi(page)
    await page.goto('/channels')

    const card = page.getByRole('heading', { name: channelName }).locator('xpath=ancestor::div[contains(@class, "hover:border-primary/30")]')
    await expect(card.getByRole('heading', { name: channelName })).toBeVisible({ timeout: 15000 })
    await card.getByRole('button').click()
    await page.getByRole('menuitem', { name: /Disable/i }).click()

    await expect
      .poll(async () => {
        const channel = await findChannelByName(request, channelName)
        return channel?.enabled
      }, { timeout: 15000 })
      .toBe(false)

    await card.getByRole('button').click()
    await page.getByRole('menuitem', { name: /Enable/i }).click()

    await expect
      .poll(async () => {
        const channel = await findChannelByName(request, channelName)
        return channel?.enabled
      }, { timeout: 15000 })
      .toBe(true)

    await deleteChannelByName(request, channelName)
  })

  const channelCreationCases: Array<{
    name: string
    heading: RegExp | string
    expectedType: string
    uniqueName?: boolean
    explicitName?: string
    fill: (dialog: import('@playwright/test').Locator, uniqueName: string) => Promise<void>
  }> = [
    {
      name: 'creates a webchat channel via UI against the real API',
      heading: 'Web Chat',
      expectedType: 'webchat',
      fill: async (dialog, uniqueName) => {
        await dialog.locator('#name').fill(uniqueName)
      },
    },
    {
      name: 'creates a WhatsApp official channel via UI against the real API',
      heading: 'WhatsApp Business',
      expectedType: 'whatsapp_official',
      fill: async (dialog, uniqueName) => {
        await dialog.getByRole('tab').nth(1).click()
        await dialog.locator('input[name="name"]').fill(uniqueName)
        await dialog.locator('input[name="access_token"]').fill('test-access-token')
        await dialog.locator('input[name="phone_number_id"]').fill('123456789012345')
      },
    },
    {
      name: 'creates a Telegram channel via UI against the real API',
      heading: 'Telegram',
      expectedType: 'telegram',
      fill: async (dialog, uniqueName) => {
        await dialog.locator('input[name="name"]').fill(uniqueName)
        await dialog.locator('input[name="bot_token"]').fill('123456:telegram-bot-token')
      },
    },
    {
      name: 'creates an SMS channel via UI against the real API',
      heading: 'SMS',
      expectedType: 'sms',
      fill: async (dialog, uniqueName) => {
        await dialog.locator('input[name="name"]').fill(uniqueName)
        await dialog.locator('input[name="account_sid"]').fill('ACtest1234567890123456789012345678')
        await dialog.locator('input[name="auth_token"]').fill('twilio-auth-token')
        await dialog.locator('input[name="phone_number"]').fill('+15551234567')
      },
    },
    {
      name: 'creates a Facebook channel via UI against the real API',
      heading: 'Facebook Messenger',
      expectedType: 'facebook',
      fill: async (dialog, uniqueName) => {
        await dialog.getByRole('tab').nth(1).click()
        await dialog.locator('input[name="name"]').fill(uniqueName)
        await dialog.locator('input[name="app_id"]').fill('facebook-app-id')
        await dialog.locator('input[name="app_secret"]').fill('facebook-app-secret')
        await dialog.locator('input[name="page_id"]').fill('facebook-page-id')
        await dialog.locator('input[name="page_access_token"]').fill('facebook-page-access-token')
      },
    },
    {
      name: 'creates an Instagram channel via UI against the real API',
      heading: 'Instagram',
      expectedType: 'instagram',
      fill: async (dialog, uniqueName) => {
        await dialog.getByRole('tab').nth(1).click()
        await dialog.locator('input[name="name"]').fill(uniqueName)
        await dialog.locator('input[name="instagram_id"]').fill('instagram-business-id')
        await dialog.locator('input[name="access_token"]').fill('instagram-access-token')
      },
    },
    {
      name: 'creates an email channel via UI against the real API',
      heading: /E-mail|Email/i,
      expectedType: 'email',
      fill: async (dialog, uniqueName) => {
        await dialog.locator('#name').fill(uniqueName)
        await dialog.locator('input[name="from_email"]').fill(`playwright.${Date.now()}@example.com`)
        await dialog.locator('input[name="smtp_host"]').fill('smtp.example.com')
        await dialog.locator('input[name="smtp_port"]').fill('587')
      },
    },
    {
      name: 'creates a WhatsApp unofficial channel via UI against the real API',
      heading: /^WhatsApp$/,
      expectedType: 'whatsapp',
      fill: async (dialog, uniqueName) => {
        await dialog.locator('input[name="name"]').fill(uniqueName)
      },
    },
    {
      name: 'creates an RCS channel via UI against the real API',
      heading: 'RCS',
      expectedType: 'rcs',
      uniqueName: false,
      explicitName: 'RCS Zenvia',
      fill: async (dialog) => {
        await dialog.locator('input[name="agentId"]').fill(`agent-${Date.now()}`)
        await dialog.locator('input[name="apiKey"]').fill('zenvia-api-token')
      },
    },
    {
      name: 'creates a voice channel via UI against the real API',
      heading: /Voice|Voz/i,
      expectedType: 'voice',
      fill: async (dialog, uniqueName) => {
        await dialog.locator('#name').fill(uniqueName)
        await dialog.locator('#account_sid').fill('ACvoice12345678901234567890123456')
        await dialog.locator('#auth_token').fill('voice-auth-token')
        await dialog.locator('#phone_number').fill('+15557654321')
      },
    },
  ]

  for (const channelCase of channelCreationCases) {
    test(channelCase.name, async ({ page, request }) => {
      await loginWithRealApi(page)
      await page.goto('/channels')

      const createdChannel = await createChannelAndGetNewRecord(page, request, {
        channelHeading: channelCase.heading,
        fill: channelCase.fill,
        expectedType: channelCase.expectedType,
        uniqueName: channelCase.uniqueName,
        explicitName: channelCase.explicitName,
      })

      await deleteChannelByID(request, createdChannel.id)
    })
  }

  test('sends a message in a conversation via UI against the real API', async ({ page, request }) => {
    const suffix = Date.now()
    const channelName = `Playwright Chat Channel ${suffix}`
    const contactName = `Playwright Contact ${suffix}`
    const contactEmail = `playwright.contact.${suffix}@example.com`
    const contactPhone = `+1555${String(suffix).slice(-7)}`
    const messageContent = `Playwright real message ${suffix}`

    await deleteChannelByName(request, channelName)
    await loginWithRealApi(page)

    const channel = await createWebchatChannelByApi(request, { name: channelName })
    const contact = await createContactByApi(request, {
      name: contactName,
      email: contactEmail,
      phone: contactPhone,
    })
    await addContactIdentityByApi(request, contact.id, {
      channelType: 'webchat',
      identifier: contactPhone,
    })
    const conversation = await createConversationByApi(request, {
      contactID: contact.id,
      channelID: channel.id,
      subject: `Playwright Subject ${suffix}`,
    })

    await page.goto('/conversations')
    await expect(page.getByRole('heading', { name: 'Conversations', level: 1 })).toBeVisible()

    const conversationItem = page.getByRole('button').filter({ hasText: /Unknown Contact|Contato desconhecido/i }).first()
    await expect(conversationItem).toBeVisible({ timeout: 15000 })
    await conversationItem.click()

    const composer = page.getByPlaceholder(/Type a message/i)
    await expect(composer).toBeVisible({ timeout: 15000 })
    await composer.fill(messageContent)
    await composer.press('Enter')

    await expect
      .poll(async () => {
        const messages = await listMessagesByConversation(request, conversation.id)
        return messages.some((message) => message.content === messageContent && message.sender_type === 'user')
      }, { timeout: 15000 })
      .toBeTruthy()

    await expect(page.getByText(messageContent)).toBeVisible({ timeout: 15000 })

    await deleteContactByID(request, contact.id)
    await deleteChannelByName(request, channelName)
  })

  test('creates a flow via UI against the real API', async ({ page, request }) => {
    const flowName = `Playwright Flow ${Date.now()}`

    await deleteFlowByName(request, flowName)
    await loginWithRealApi(page)
    await page.goto('/flows')

    await page.getByRole('button', { name: /New Flow/i }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await dialog.getByLabel('Name').fill(flowName)
    await dialog.getByLabel(/Description/i).fill('Flow created by Playwright full-stack test')
    await dialog.getByRole('button', { name: /Create & Edit/i }).click()

    await expect
      .poll(async () => {
        const flow = await findFlowByName(request, flowName)
        return flow?.trigger || null
      }, { timeout: 15000 })
      .toBe('keyword')

    const createdFlow = await findFlowByName(request, flowName)
    expect(createdFlow?.is_active).toBe(false)

    await deleteFlowByName(request, flowName)
  })

  test('activates a flow via UI against the real API', async ({ page, request }) => {
    const flowName = `Playwright Activate Flow ${Date.now()}`

    await deleteFlowByName(request, flowName)
    await loginWithRealApi(page)

    const accessToken = await loginAsAdminApi(request)
    const createResponse = await request.post('http://localhost:8081/api/v1/flows', {
      headers: {
        Authorization: `Bearer ${accessToken}`,
      },
      data: {
        name: flowName,
        description: 'Flow activation test',
        trigger: 'manual',
        start_node_id: 'start',
        nodes: [
          {
            id: 'start',
            type: 'message',
            content: 'Hello from Playwright',
            transitions: [],
            position: { x: 250, y: 50 },
          },
        ],
      },
    })
    expect(createResponse.ok()).toBeTruthy()

    await page.goto('/flows')

    const card = page.getByRole('heading', { name: flowName }).locator('xpath=ancestor::div[contains(@class, "group relative")]')
    await expect(card.getByRole('heading', { name: flowName })).toBeVisible({ timeout: 15000 })
    await card.getByRole('button').first().click()
    await page.getByRole('menuitem', { name: /Activate/i }).click()

    await expect
      .poll(async () => {
        const flow = await findFlowByName(request, flowName)
        return flow?.is_active ?? null
      }, { timeout: 15000 })
      .toBe(true)

    await deleteFlowByName(request, flowName)
  })
})
