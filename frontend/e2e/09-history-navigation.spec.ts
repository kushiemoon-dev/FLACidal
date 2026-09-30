import { test, expect } from '@playwright/test'
import { injectWailsMocks, type WinAny } from './mocks/wails'

const heading = (page: import('@playwright/test').Page, name: RegExp) =>
  page.locator('h1, h2').filter({ hasText: name }).first()

test.describe('Back / Forward / Reload navigation', () => {
  test.beforeEach(async ({ page }) => {
    await injectWailsMocks(page)
    await page.goto('/')
    await expect(page.locator('.sidebar')).toBeVisible()
  })

  test('Back and Forward start disabled', async ({ page }) => {
    await expect(page.locator('.sidebar button[title="Back"]')).toBeDisabled()
    await expect(page.locator('.sidebar button[title="Forward"]')).toBeDisabled()
  })

  test('Alt+Arrow keys walk the page history', async ({ page }) => {
    await page.locator('.sidebar button[title="Settings"]').click()
    await expect(heading(page, /Settings/i)).toBeVisible()
    await page.locator('.sidebar button[title="Terminal"]').click()
    await expect(heading(page, /Terminal/i)).toBeVisible()

    await page.keyboard.press('Alt+ArrowLeft')
    await expect(heading(page, /Settings/i)).toBeVisible()
    await page.keyboard.press('Alt+ArrowRight')
    await expect(heading(page, /Terminal/i)).toBeVisible()
  })

  test('navigating after Back clears Forward', async ({ page }) => {
    await page.locator('.sidebar button[title="Settings"]').click()
    await page.locator('.sidebar button[title="Terminal"]').click()
    await page.locator('.sidebar button[title="Back"]').click()
    await expect(page.locator('.sidebar button[title="Forward"]')).toBeEnabled()
    await page.locator('.sidebar button[title="About"]').click()
    await expect(page.locator('.sidebar button[title="Forward"]')).toBeDisabled()
  })

  test('F5 and Alt+Left never trigger a native reload or Back', async ({ page }) => {
    await page.evaluate(() => { (window as unknown as WinAny).__marker = 1 })
    const url = page.url()

    await page.keyboard.press('F5')
    await page.keyboard.press('Alt+ArrowLeft')
    await page.waitForTimeout(300)

    expect(page.url()).toBe(url)
    expect(await page.evaluate(() => (window as unknown as WinAny).__marker)).toBe(1)
  })

  test('Alt+Arrow inside a text field does not change page', async ({ page }) => {
    await page.locator('.sidebar button[title="Settings"]').click()
    await page.locator('.sidebar button[title="Search"]').click()
    const input = page.locator('.main-content input').first()
    await input.click()
    await page.keyboard.press('Alt+ArrowLeft')
    await expect(heading(page, /Search/i)).toBeVisible()
    await expect(page.locator('.sidebar button[title="Forward"]')).toBeDisabled()
  })
})
