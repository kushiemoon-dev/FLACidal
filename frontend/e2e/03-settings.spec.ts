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
  test('Tidal HiFi Priority Instances field is present', async ({ page }) => {
    await gotoSettings(page)
    await expect(
      page.locator('.setting-label', { hasText: 'Tidal HiFi Priority Instances' }).first(),
    ).toBeVisible()
  })

  test('Qobuz Priority Instances field is present', async ({ page }) => {
    await gotoSettings(page)
    await expect(
      page.locator('.setting-label', { hasText: 'Qobuz Priority Instances' }).first(),
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

  test('typing into Tidal priority instances textarea persists', async ({ page }) => {
    await gotoSettings(page)
    const tidalTextarea = page.locator('textarea.endpoint-list').first()
    await tidalTextarea.scrollIntoViewIfNeeded()
    await tidalTextarea.fill('https://my-tidal-api.example.com')
    await expect(tidalTextarea).toHaveValue('https://my-tidal-api.example.com')
  })

  test('typing into Qobuz priority instances textarea persists', async ({ page }) => {
    await gotoSettings(page)
    const qobuzTextarea = page.locator('textarea.endpoint-list').nth(1)
    await qobuzTextarea.scrollIntoViewIfNeeded()
    await qobuzTextarea.fill('https://my-qobuz.example.com')
    await expect(qobuzTextarea).toHaveValue('https://my-qobuz.example.com')
  })
})
