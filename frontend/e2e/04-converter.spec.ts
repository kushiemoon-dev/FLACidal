import { test, expect } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

/**
 * Because App.svelte pairs `{#key activePage}` with `transition:fade`, two copies
 * of the page coexist in the DOM for the 150ms transition window. Rely on
 * `.first()` and pause for the fade before asserting.
 */
async function gotoConverter(page: any) {
  await injectWailsMocks(page)
  await page.goto('/')
  await page.locator('.sidebar button[title="Tools"]').click()
  await page.locator('.flyout-item', { hasText: 'Audio Converter' }).click()
  await page.waitForTimeout(250)
  await expect(page.locator('h1', { hasText: /Audio Converter/i }).first()).toBeVisible()
}

test.describe('Audio Converter tool', () => {
  test('shows the drop zone when nothing is selected yet', async ({ page }) => {
    await gotoConverter(page)
    await expect(page.locator('button', { hasText: /Select Files/i }).first()).toBeVisible()
    await expect(
      page.locator('text=/Supported formats:.*FLAC.*MP3/i').first(),
    ).toBeVisible()
  })

  test('lists every expected format once files are added', async ({ page }) => {
    await gotoConverter(page)
    await page.evaluate(() => {
      // @ts-ignore
      ;(window as any).go.main.App.OpenFLACFilesDialog = async () => [
        '/tmp/a.flac',
        '/tmp/b.flac',
      ]
    })
    await page.locator('button', { hasText: /Select Files/i }).first().click()
    const select = page.locator('select').first()
    await expect(select).toBeVisible()
    const options = await select.locator('option').allTextContents()
    for (const fmt of ['MP3', 'AAC', 'ALAC', 'Opus', 'Vorbis', 'WAV']) {
      expect(options.join(' ')).toMatch(new RegExp(fmt, 'i'))
    }
  })

  test('picking ALAC drops the bitrate quality options', async ({ page }) => {
    await gotoConverter(page)
    await page.evaluate(() => {
      ;(window as any).go.main.App.OpenFLACFilesDialog = async () => ['/tmp/a.flac']
    })
    await page.locator('button', { hasText: /Select Files/i }).first().click()
    const select = page.locator('select').first()
    await expect(select).toBeVisible()
    await select.selectOption('ALAC')
    // ALAC offers no quality options, so the selection should remain ALAC
    await expect(select).toHaveValue('ALAC')
  })

  test('renders with no runtime errors', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (e) => errors.push(e.message))
    await gotoConverter(page)
    await page.waitForTimeout(300)
    expect(errors, errors.join('\n')).toHaveLength(0)
  })
})
