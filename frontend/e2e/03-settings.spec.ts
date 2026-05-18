import { test, expect } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

/**
 * Note: App.svelte wraps page content in `{#key activePage} <div transition:fade>`,
 * which keeps both the previous and next page in the DOM during the 150ms fade.
 * Several Settings labels therefore appear twice transiently. We always use `.first()`
 * and wait for the transition to settle.
 */
async function gotoSettings(page: any) {
  await injectWailsMocks(page)
  await page.goto('/')
  await page.locator('.sidebar button[title="Settings"]').click()
  // wait for fade to finish (>150ms) before strict locators
  await page.waitForTimeout(250)
  await expect(page.locator('h1').filter({ hasText: /^Settings$/ }).first()).toBeVisible()
}

test.describe('Settings page', () => {
  test('Custom Tidal HiFi API URL field is present', async ({ page }) => {
    await gotoSettings(page)
    await expect(
      page.locator('.setting-label', { hasText: 'Custom Tidal HiFi API URL' }).first(),
    ).toBeVisible()
  })

  test('Custom Qobuz API URL field is present', async ({ page }) => {
    await gotoSettings(page)
    await expect(
      page.locator('.setting-label', { hasText: 'Custom Qobuz API URL' }).first(),
    ).toBeVisible()
  })

  test('Save Changes button exists', async ({ page }) => {
    await gotoSettings(page)
    await expect(page.locator('button', { hasText: /Save Changes/i }).first()).toBeVisible()
  })

  test('Source toggle (Tidal) is rendered', async ({ page }) => {
    await gotoSettings(page)
    await expect(page.locator('input[type="checkbox"]').first()).toBeAttached()
  })

  test('typing into custom Tidal endpoint persists in input', async ({ page }) => {
    await gotoSettings(page)
    const tidalInput = page
      .locator('input[placeholder="https://your-hifi-api-instance.com"]')
      .first()
    await tidalInput.scrollIntoViewIfNeeded()
    await tidalInput.fill('https://my-tidal-api.example.com')
    await expect(tidalInput).toHaveValue('https://my-tidal-api.example.com')
  })

  test('typing into custom Qobuz endpoint persists in input', async ({ page }) => {
    await gotoSettings(page)
    const qobuzInput = page
      .locator('input[placeholder="https://your-qobuz-proxy.com"]')
      .first()
    await qobuzInput.scrollIntoViewIfNeeded()
    await qobuzInput.fill('https://my-qobuz.example.com')
    await expect(qobuzInput).toHaveValue('https://my-qobuz.example.com')
  })
})
