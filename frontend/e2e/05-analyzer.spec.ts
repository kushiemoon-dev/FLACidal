import { test, expect } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

/**
 * The standalone `pages/Analyzer.svelte` (which posts to /api/analyze) is NOT
 * routed in App.svelte. The user-facing analyzer is `tools/AudioQualityAnalyzer.svelte`,
 * which calls the Wails binding `AnalyzeMultiple`. We test that page here.
 *
 * App.svelte's `{#key activePage}` + transition:fade keeps two DOM copies during
 * the 150ms fade — always use `.first()`.
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
  test('page loads via tools flyout', async ({ page }) => {
    await gotoAnalyzer(page)
  })

  test('drop zone or select-files affordance is present', async ({ page }) => {
    await gotoAnalyzer(page)
    await expect(
      page.locator('text=/Drop|Drag|FLAC|Select|Choose/i').first(),
    ).toBeVisible()
  })

  test('renders without runtime errors', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (e) => errors.push(e.message))
    await gotoAnalyzer(page)
    await page.waitForTimeout(300)
    expect(errors, errors.join('\n')).toHaveLength(0)
  })
})
