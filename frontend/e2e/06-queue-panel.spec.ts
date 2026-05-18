import { test, expect } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

test.describe('QueuePanel (global, mounted in App.svelte)', () => {
  test('panel is mounted and visible', async ({ page }) => {
    await injectWailsMocks(page)
    await page.goto('/')
    await expect(page.locator('.queue-panel')).toBeVisible()
  })

  test('shows empty state when no jobs', async ({ page }) => {
    await injectWailsMocks(page)
    await page.goto('/')
    // Expand if collapsed (panel-header click toggles)
    const panel = page.locator('.queue-panel')
    await expect(panel).toBeVisible()
    // The empty message uses literal French text from QueuePanel.svelte
    // "Aucun téléchargement en cours" — only visible when expanded.
    // First make sure the body is open by clicking the header if needed:
    const collapsed = await panel.evaluate((el) => el.classList.contains('collapsed'))
    if (collapsed) {
      await panel.locator('.panel-header').click()
    }
    await expect(panel.locator('.empty')).toBeVisible()
    await expect(panel.locator('.empty')).toContainText(/Aucun téléchargement|No download/i)
  })

  test('does not crash when websocket unavailable', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (e) => errors.push(e.message))
    await injectWailsMocks(page)
    await page.goto('/')
    await page.waitForTimeout(500)
    expect(errors, errors.join('\n')).toHaveLength(0)
  })
})
