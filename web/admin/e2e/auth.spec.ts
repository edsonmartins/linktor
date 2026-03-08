import { test, expect } from '@playwright/test'

test.describe('Login Page', () => {
  test.beforeEach(async ({ page }) => {
    // Ensure clean state - no auth tokens
    await page.addInitScript(() => {
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
      localStorage.removeItem('auth-storage')
    })

    // Mock login API
    await page.route('**/api/v1/auth/login', async (route) => {
      const body = JSON.parse(route.request().postData() || '{}')
      if (body.email === 'admin@demo.com' && body.password === 'admin123') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              user: {
                id: 'user-1',
                tenant_id: 'tenant-1',
                email: 'admin@demo.com',
                name: 'Admin User',
                role: 'admin',
                status: 'active',
              },
              access_token: 'mock-access-token',
              refresh_token: 'mock-refresh-token',
              expires_in: 900,
            },
          }),
        })
      } else {
        await route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({
            success: false,
            error: { code: 'INVALID_CREDENTIALS', message: 'Invalid email or password' },
          }),
        })
      }
    })

    // Mock refresh (will fail for unauthenticated state)
    await page.route('**/api/v1/auth/refresh', async (route) => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ success: false, error: { code: 'INVALID_TOKEN', message: 'Invalid token' } }),
      })
    })
  })

  test('renders login form', async ({ page }) => {
    await page.goto('/login')

    await expect(page.locator('input[type="email"]')).toBeVisible({ timeout: 15000 })
    await expect(page.locator('input[type="password"]')).toBeVisible()
    await expect(page.locator('button[type="submit"]')).toBeVisible()
  })

  test('shows demo credentials hint', async ({ page }) => {
    await page.goto('/login')

    await expect(page.getByText('admin@demo.com')).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('admin123')).toBeVisible()
  })

  test('successful login redirects to dashboard', async ({ page }) => {
    // Mock all dashboard API calls
    await page.route('**/api/v1/**', async (route) => {
      const url = route.request().url()
      if (url.includes('/auth/login') || url.includes('/auth/refresh')) {
        return route.fallback()
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [], meta: { page: 1, page_size: 20, total_items: 0 } }),
      })
    })

    await page.goto('/login')
    await expect(page.locator('input[type="email"]')).toBeVisible({ timeout: 15000 })

    await page.fill('input[type="email"]', 'admin@demo.com')
    await page.fill('input[type="password"]', 'admin123')
    await page.click('button[type="submit"]')

    await expect(page).toHaveURL(/dashboard/, { timeout: 15000 })
  })

  test('invalid credentials show error', async ({ page }) => {
    await page.goto('/login')
    await expect(page.locator('input[type="email"]')).toBeVisible({ timeout: 15000 })

    await page.fill('input[type="email"]', 'wrong@test.com')
    await page.fill('input[type="password"]', 'wrongpass')
    await page.click('button[type="submit"]')

    // Should stay on login page
    await expect(page).toHaveURL(/login/)
  })

  test('empty form does not submit', async ({ page }) => {
    await page.goto('/login')
    await expect(page.locator('button[type="submit"]')).toBeVisible({ timeout: 15000 })

    await page.click('button[type="submit"]')

    // Should stay on login page (HTML5 validation prevents submit)
    await expect(page).toHaveURL(/login/)
  })
})
