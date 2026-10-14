import { test, expect, type Page } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

/**
 * Heads up: App.svelte nests page content inside `{#key activePage} <div transition:fade>`,
 * so both the outgoing and incoming pages stay in the DOM for the 150ms fade window.
 * That means a handful of Settings labels show up twice for a moment, always reach
 * for `.first()` and give the transition time to finish before asserting.
 */
async function gotoSettings(page: Page) {
  await injectWailsMocks(page)
  await page.goto('/')
  await page.locator('.sidebar button[title="Settings"]').click()
  // give the fade transition (150ms) time to finish before using strict locators
  await page.waitForTimeout(250)
  await expect(page.locator('h1').filter({ hasText: /^Settings$/ }).first()).toBeVisible()
}

async function openAdvancedInstances(page: Page) {
  await page.locator('details.advanced-instances > summary').click()
}

test.describe('Settings status tab', () => {
  test('shows a never-tested endpoint as untested, not online', async ({ page }) => {
    await injectWailsMocks(page, {
      GetSourceHealth: [
        {
          name: 'tidal',
          displayName: 'Tidal HiFi',
          status: 'untested',
          latencyMs: 0,
          endpoints: [{ url: 'https://proxy.example.com', state: 'untested', fails: 0, revivals: 0 }],
        },
      ],
    })
    await page.goto('/')
    await page.locator('.sidebar button[title="Settings"]').click()
    await page.waitForTimeout(250)
    await page.locator('.tab-item', { hasText: /Status/i }).first().click()

    await expect(page.locator('button', { hasText: /Test endpoints/i }).first()).toBeVisible()
    const row = page.locator('.endpoint-health-row').first()
    await expect(row.locator('.status-badge')).toHaveClass(/untested/)
    await expect(row.locator('.status-badge')).not.toHaveClass(/\bok\b/)
  })
})

test.describe('Settings status tab probe results', () => {
  test('shows the raw failure and the reference-not-found note', async ({ page }) => {
    await injectWailsMocks(page, {
      GetSourceHealth: [
        {
          name: 'tidal',
          displayName: 'Tidal HiFi',
          status: 'dead',
          latencyMs: 0,
          endpoints: [
            { url: 'https://down.example.com', state: 'untested', fails: 0, revivals: 0, probe: { kind: 'failed', message: 'upstream api error', at: '2026-10-14T12:00:00Z' } },
            { url: 'https://gap.example.com', state: 'untested', fails: 0, revivals: 0, probe: { kind: 'notfound', at: '2026-10-14T12:00:00Z' } },
          ],
        },
      ],
    })
    await page.goto('/')
    await page.locator('.sidebar button[title="Settings"]').click()
    await page.waitForTimeout(250)
    await page.locator('.tab-item', { hasText: /Status/i }).first().click()

    const rows = page.locator('.endpoint-health-row .status-badge')
    await expect(rows.nth(0)).toHaveClass(/error/)
    await expect(page.getByText('upstream api error')).toBeVisible()
    await expect(rows.nth(1)).toHaveClass(/untested/)
    await expect(page.getByText('reference track not found')).toBeVisible()
  })
})

test.describe('Settings screen', () => {
  test('renders the Tidal HiFi instances field', async ({ page }) => {
    await gotoSettings(page)
    await openAdvancedInstances(page)
    await expect(
      page.locator('.advanced-instances .setting-label', { hasText: 'Tidal HiFi' }).first(),
    ).toBeVisible()
  })

  test('renders the Qobuz instances field', async ({ page }) => {
    await gotoSettings(page)
    await openAdvancedInstances(page)
    await expect(
      page.locator('.advanced-instances .setting-label', { hasText: 'Qobuz' }).first(),
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

  test('keeps what you type into the Tidal instances textarea', async ({ page }) => {
    await gotoSettings(page)
    await openAdvancedInstances(page)
    const tidalTextarea = page.locator('.advanced-instances textarea.endpoint-list').first()
    await tidalTextarea.scrollIntoViewIfNeeded()
    await tidalTextarea.fill('https://my-tidal-api.example.com')
    await expect(tidalTextarea).toHaveValue('https://my-tidal-api.example.com')
  })

  test('keeps what you type into the Qobuz instances textarea', async ({ page }) => {
    await gotoSettings(page)
    await openAdvancedInstances(page)
    const qobuzTextarea = page.locator('.advanced-instances textarea.endpoint-list').nth(1)
    await qobuzTextarea.scrollIntoViewIfNeeded()
    await qobuzTextarea.fill('https://my-qobuz.example.com')
    await expect(qobuzTextarea).toHaveValue('https://my-qobuz.example.com')
  })
})
