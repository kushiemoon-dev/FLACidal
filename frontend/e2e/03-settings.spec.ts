import { test, expect } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

/**
 * Heads up: App.svelte nests page content inside `{#key activePage} <div transition:fade>`,
 * so both the outgoing and incoming pages stay in the DOM for the 150ms fade window.
 * That means a handful of Settings labels show up twice for a moment, always reach
 * for `.first()` and give the transition time to finish before asserting.
 */
async function gotoSettings(page: any) {
  await injectWailsMocks(page)
  await page.goto('/')
  await page.locator('.sidebar button[title="Settings"]').click()
  // give the fade transition (150ms) time to finish before using strict locators
  await page.waitForTimeout(250)
  await expect(page.locator('h1').filter({ hasText: /^Settings$/ }).first()).toBeVisible()
}

test.describe('Settings screen', () => {
  test('renders the Tidal HiFi Priority Instances field', async ({ page }) => {
    await gotoSettings(page)
    await expect(
      page.locator('.setting-label', { hasText: 'Tidal HiFi Priority Instances' }).first(),
    ).toBeVisible()
  })

  test('renders the Qobuz Priority Instances field', async ({ page }) => {
    await gotoSettings(page)
    await expect(
      page.locator('.setting-label', { hasText: 'Qobuz Priority Instances' }).first(),
    ).toBeVisible()
  })

  test('shows a Save Changes button', async ({ page }) => {
    await gotoSettings(page)
    await expect(page.locator('button', { hasText: /Save Changes/i }).first()).toBeVisible()
  })

  test('renders the Tidal source toggle', async ({ page }) => {
    await gotoSettings(page)
    await expect(page.locator('input[type="checkbox"]').first()).toBeAttached()
  })

  test('keeps what you type into the Tidal priority instances textarea', async ({ page }) => {
    await gotoSettings(page)
    const tidalTextarea = page.locator('textarea.endpoint-list').first()
    await tidalTextarea.scrollIntoViewIfNeeded()
    await tidalTextarea.fill('https://my-tidal-api.example.com')
    await expect(tidalTextarea).toHaveValue('https://my-tidal-api.example.com')
  })

  test('keeps what you type into the Qobuz priority instances textarea', async ({ page }) => {
    await gotoSettings(page)
    const qobuzTextarea = page.locator('textarea.endpoint-list').nth(1)
    await qobuzTextarea.scrollIntoViewIfNeeded()
    await qobuzTextarea.fill('https://my-qobuz.example.com')
    await expect(qobuzTextarea).toHaveValue('https://my-qobuz.example.com')
  })
})
