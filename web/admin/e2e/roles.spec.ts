import { expect, test } from '@playwright/test'
import { mockAPIError, mockEmptyEndpoints, setupAuth } from './helpers'

const CATALOG = {
  resources: ['contacts', 'conversations', 'channels', 'roles'],
  actions: ['create', 'read', 'update', 'delete'],
}

const ALL_PERMISSIONS = CATALOG.resources.flatMap((resource) =>
  CATALOG.actions.map((action) => ({ resource, action }))
)

const SYSTEM_ROLE = {
  id: 'role-1',
  tenant_id: 'tenant-1',
  name: 'Administrator',
  description: 'Full access to everything',
  is_system: true,
  is_default: false,
  permissions: ALL_PERMISSIONS,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const CUSTOM_ROLE = {
  id: 'role-2',
  tenant_id: 'tenant-1',
  name: 'Support Agent',
  description: 'Read-only support access',
  is_system: false,
  is_default: true,
  permissions: [
    { resource: 'contacts', action: 'read' },
    { resource: 'conversations', action: 'read' },
  ],
  created_at: '2026-02-01T00:00:00Z',
  updated_at: '2026-02-01T00:00:00Z',
}

async function mockRolesEndpoints(page: import('@playwright/test').Page) {
  await mockEmptyEndpoints(page)

  // Generic single-role handler first: more specific routes registered after
  // it take precedence (Playwright matches the most recent registration).
  await page.route('**/api/v1/roles/*', async (route) => {
    const method = route.request().method()
    if (method === 'PUT') {
      const body = route.request().postDataJSON() as Record<string, unknown>
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { ...CUSTOM_ROLE, ...body } }),
      })
      return
    }
    if (method === 'DELETE') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: null }),
      })
      return
    }
    await route.fallback()
  })

  await page.route('**/api/v1/roles/catalog', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: CATALOG }),
    })
  })

  await page.route('**/api/v1/roles', async (route) => {
    if (route.request().method() === 'POST') {
      const body = route.request().postDataJSON() as Record<string, unknown>
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            id: 'role-3',
            tenant_id: 'tenant-1',
            is_system: false,
            is_default: false,
            created_at: '2026-06-01T00:00:00Z',
            updated_at: '2026-06-01T00:00:00Z',
            ...body,
          },
        }),
      })
      return
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: [SYSTEM_ROLE, CUSTOM_ROLE] }),
    })
  })
}

test.describe('Roles Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await mockRolesEndpoints(page)
  })

  test('lists system and custom roles with badges', async ({ page }) => {
    await page.goto('/roles')

    await expect(page.getByRole('heading', { name: 'Roles' })).toBeVisible({ timeout: 15000 })

    const systemRow = page.getByRole('row', { name: 'Administrator' })
    await expect(systemRow.getByText('System')).toBeVisible()
    await expect(systemRow.getByText('Full access to everything')).toBeVisible()

    const customRow = page.getByRole('row', { name: 'Support Agent' })
    await expect(customRow.getByText('Default')).toBeVisible()
    await expect(customRow.getByText('Read-only support access')).toBeVisible()
  })

  test('shows permission counts per role', async ({ page }) => {
    await page.goto('/roles')

    await expect(page.getByRole('heading', { name: 'Roles' })).toBeVisible({ timeout: 15000 })

    await expect(
      page.getByRole('row', { name: 'Administrator' }).getByRole('cell', { name: '16', exact: true })
    ).toBeVisible()
    await expect(
      page.getByRole('row', { name: 'Support Agent' }).getByRole('cell', { name: '2', exact: true })
    ).toBeVisible()
  })

  test('creates a custom role with selected permissions', async ({ page }) => {
    let createPayload: Record<string, unknown> | null = null

    await page.route('**/api/v1/roles', async (route) => {
      if (route.request().method() !== 'POST') {
        await route.fallback()
        return
      }
      createPayload = route.request().postDataJSON() as Record<string, unknown>
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            id: 'role-3',
            tenant_id: 'tenant-1',
            is_system: false,
            is_default: false,
            ...createPayload,
          },
        }),
      })
    })

    await page.goto('/roles')

    await expect(page.getByRole('heading', { name: 'Roles' })).toBeVisible({ timeout: 15000 })
    await page.getByRole('button', { name: 'New role' }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog.getByRole('heading', { name: 'New role' })).toBeVisible()

    // Inputs are not label-associated (no htmlFor), so target by position:
    // first textbox = Name, second = Description.
    await dialog.getByRole('textbox').first().fill('Quality Auditor')
    await dialog.getByRole('textbox').nth(1).fill('Read-only QA access')

    // Actions are columns in catalog order: create(0), read(1), update(2), delete(3)
    await dialog.getByRole('row', { name: 'contacts' }).getByRole('checkbox').nth(1).check()
    await dialog.getByRole('row', { name: 'conversations' }).getByRole('checkbox').nth(1).check()

    await dialog.getByRole('button', { name: 'Create role' }).click()

    await expect.poll(() => createPayload).not.toBeNull()
    expect(createPayload).toMatchObject({
      name: 'Quality Auditor',
      description: 'Read-only QA access',
      permissions: expect.arrayContaining([
        { resource: 'contacts', action: 'read' },
        { resource: 'conversations', action: 'read' },
      ]),
    })

    await expect(dialog).toBeHidden()
  })

  test('edits permissions of a custom role', async ({ page }) => {
    let putUrl: string | null = null
    let putPayload: Record<string, unknown> | null = null

    await page.route('**/api/v1/roles/*', async (route) => {
      if (route.request().method() !== 'PUT') {
        await route.fallback()
        return
      }
      putUrl = route.request().url()
      putPayload = route.request().postDataJSON() as Record<string, unknown>
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { ...CUSTOM_ROLE, ...putPayload } }),
      })
    })

    await page.goto('/roles')

    await expect(page.getByRole('heading', { name: 'Roles' })).toBeVisible({ timeout: 15000 })

    // Pencil button on a custom role gets accessible name "Save" via title attr.
    await page.getByRole('row', { name: 'Support Agent' }).getByRole('button', { name: 'Save' }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog.getByRole('heading', { name: 'Edit role' })).toBeVisible()
    await expect(dialog.getByRole('textbox').first()).toHaveValue('Support Agent')

    // Existing permission contacts:read should come pre-checked.
    const contactsRow = dialog.getByRole('row', { name: 'contacts' })
    await expect(contactsRow.getByRole('checkbox').nth(1)).toBeChecked()

    // Grant contacts:update on top of the existing permissions.
    await contactsRow.getByRole('checkbox').nth(2).check()

    await dialog.getByRole('button', { name: 'Save', exact: true }).click()

    await expect.poll(() => putUrl).not.toBeNull()
    expect(putUrl).toContain('/api/v1/roles/role-2')
    expect(putPayload).toMatchObject({
      name: 'Support Agent',
      permissions: expect.arrayContaining([
        { resource: 'contacts', action: 'read' },
        { resource: 'contacts', action: 'update' },
        { resource: 'conversations', action: 'read' },
      ]),
    })

    await expect(dialog).toBeHidden()
  })

  test('system role opens read-only without save button', async ({ page }) => {
    await page.goto('/roles')

    await expect(page.getByRole('heading', { name: 'Roles' })).toBeVisible({ timeout: 15000 })

    await page
      .getByRole('row', { name: 'Administrator' })
      .getByRole('button', { name: 'System roles cannot be edited' })
      .click()

    const dialog = page.getByRole('dialog')
    await expect(dialog.getByText('System roles cannot be edited')).toBeVisible()
    await expect(dialog.getByRole('textbox').first()).toBeDisabled()
    await expect(dialog.getByRole('button', { name: 'Save', exact: true })).toHaveCount(0)

    await dialog.getByRole('button', { name: 'Cancel' }).click()
    await expect(dialog).toBeHidden()
  })

  test('deletes a custom role after confirmation', async ({ page }) => {
    let deleteUrl: string | null = null

    await page.route('**/api/v1/roles/*', async (route) => {
      if (route.request().method() !== 'DELETE') {
        await route.fallback()
        return
      }
      deleteUrl = route.request().url()
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: null }),
      })
    })

    await page.goto('/roles')

    await expect(page.getByRole('heading', { name: 'Roles' })).toBeVisible({ timeout: 15000 })

    // Custom role row has two icon buttons: pencil ("Save") then trash (no name).
    await page.getByRole('row', { name: 'Support Agent' }).getByRole('button').last().click()

    const confirm = page.getByRole('alertdialog')
    await expect(confirm.getByText('Delete this role?')).toBeVisible()
    await confirm.getByRole('button', { name: 'Delete' }).click()

    await expect.poll(() => deleteUrl).not.toBeNull()
    expect(deleteUrl).toContain('/api/v1/roles/role-2')

    await expect(confirm).toBeHidden()
  })

  test('shows error toast when delete fails', async ({ page }) => {
    // Registered last, takes precedence over the success handlers for /roles/*.
    await mockAPIError(page, 'roles/*', 500, 'Cannot delete role')

    await page.goto('/roles')

    await expect(page.getByRole('heading', { name: 'Roles' })).toBeVisible({ timeout: 15000 })

    await page.getByRole('row', { name: 'Support Agent' }).getByRole('button').last().click()

    const confirm = page.getByRole('alertdialog')
    await expect(confirm.getByText('Delete this role?')).toBeVisible()
    await confirm.getByRole('button', { name: 'Delete' }).click()

    await expect(page.getByText('Cannot delete role').first()).toBeVisible()
  })
})
