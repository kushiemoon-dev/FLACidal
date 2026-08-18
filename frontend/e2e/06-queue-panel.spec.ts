import { test, expect } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

test.describe('QueuePanel (global component mounted from App.svelte)', () => {
  test('is mounted and visible on load', async ({ page }) => {
    await injectWailsMocks(page)
    await page.goto('/')
    await expect(page.locator('.queue-panel')).toBeVisible()
  })

  test('shows the empty state when there are no jobs', async ({ page }) => {
    await injectWailsMocks(page)
    await page.goto('/')
    const panel = page.locator('.queue-panel')
    await expect(panel).toBeVisible()
    // QueuePanel.svelte renders the exact text "No downloads in progress" for
    // its empty state, but that copy is only visible once the panel is expanded.
    // Make sure the body is open before checking for it:
    const collapsed = await panel.evaluate((el) => el.classList.contains('collapsed'))
    if (collapsed) {
      await panel.locator('.panel-header').click()
    }
    await expect(panel.locator('.empty')).toBeVisible()
    await expect(panel.locator('.empty')).toContainText(/No download/i)
  })

  test('stays up even when the websocket is unavailable', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (e) => errors.push(e.message))
    await injectWailsMocks(page)
    await page.goto('/')
    await page.waitForTimeout(500)
    expect(errors, errors.join('\n')).toHaveLength(0)
  })
})
