import { test, expect } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

/**
 * `pages/Analyzer.svelte` (which posts to /api/analyze) is a standalone page and
 * isn't wired into App.svelte's routing. What users actually reach is
 * `tools/AudioQualityAnalyzer.svelte`, which talks to the Wails binding
 * `AnalyzeMultiple`, that's the page exercised below.
 *
 * As elsewhere, App.svelte's `{#key activePage}` + transition:fade leaves two
 * copies in the DOM for the fade duration, so lean on `.first()`.
 */
async function gotoAnalyzer(page: any) {
  await injectWailsMocks(page)
  await page.goto('/')
  await page.locator('.sidebar button[title="Tools"]').click()
  await page.locator('.flyout-item', { hasText: 'Audio Quality Analyzer' }).click()
  await page.waitForTimeout(250)
  await expect(page.locator('h1', { hasText: /Analyzer/i }).first()).toBeVisible()
}

test.describe('Audio Quality Analyzer tool', () => {
  test('opens successfully from the tools flyout', async ({ page }) => {
    await gotoAnalyzer(page)
  })

  test('offers a way to drop or select files', async ({ page }) => {
    await gotoAnalyzer(page)
    await expect(
      page.locator('text=/Drop|Drag|FLAC|Select|Choose/i').first(),
    ).toBeVisible()
  })

  test('loads with no runtime errors', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (e) => errors.push(e.message))
    await gotoAnalyzer(page)
    await page.waitForTimeout(300)
    expect(errors, errors.join('\n')).toHaveLength(0)
  })
})
