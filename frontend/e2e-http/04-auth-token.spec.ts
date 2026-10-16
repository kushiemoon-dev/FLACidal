import { test, expect } from '@playwright/test'
import { TOKEN, TOKEN_URL } from './servers'

test.use({ baseURL: TOKEN_URL })

test.describe('Access token on the real server', () => {
  test('asks for the token, rejects a wrong one, accepts the right one', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('.auth-required')).toBeVisible()

    await page.locator('.auth-input').fill('wrong-token')
    await page.locator('.btn-unlock').click()
    await expect(page.locator('[data-testid="auth-rejected"]')).toBeVisible()

    await page.locator('.auth-input').fill(TOKEN)
    await page.locator('.btn-unlock').click()
    await expect(page.locator('.sidebar')).toBeVisible()
  })

  test('health is open, the rest of the API needs the token', async ({ request }) => {
    expect((await request.get('/api/health')).status()).toBe(200)
    expect((await request.get('/api/version')).status()).toBe(401)
    const ok = await request.get('/api/version', { headers: { Authorization: `Bearer ${TOKEN}` } })
    expect(ok.status()).toBe(200)
  })
})
