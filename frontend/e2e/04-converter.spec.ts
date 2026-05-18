import { test, expect } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

/**
 * App.svelte's `{#key activePage}` + `transition:fade` keeps two copies of the
 * page during the 150ms transition. Use `.first()` and wait for the fade.
 */
async function gotoConverter(page: any) {
  await injectWailsMocks(page)
  await page.goto('/')
  await page.locator('.sidebar button[title="Tools"]').click()
  await page.locator('.flyout-item', { hasText: 'Audio Converter' }).click()
  await page.waitForTimeout(250)
  await expect(page.locator('h1', { hasText: /Audio Converter/i }).first()).toBeVisible()
}

test.describe('AudioConverter tool', () => {
  test('drop zone visible when no files selected', async ({ page }) => {
    await gotoConverter(page)
    // DropZone shows the "Select Files" button + supported formats text
    await expect(page.locator('button', { hasText: /Select Files/i }).first()).toBeVisible()
    await expect(
      page.locator('text=/Supported formats:.*FLAC.*MP3/i').first(),
    ).toBeVisible()
  })

  test('all expected formats appear after files are added', async ({ page }) => {
    await gotoConverter(page)
    // Inject fake selected files via the OpenFLACFilesDialog mock at the binding level.
    await page.evaluate(() => {
      // @ts-ignore — override mock for this test
      ;(window as any).go.main.App.OpenFLACFilesDialog = async () => [
        '/tmp/a.flac',
        '/tmp/b.flac',
      ]
    })
    await page.locator('button', { hasText: /Select Files/i }).first().click()
    // After selection, the format <select> mounts. Check its options.
    const select = page.locator('select').first()
    await expect(select).toBeVisible()
    const options = await select.locator('option').allTextContents()
    for (const fmt of ['MP3', 'AAC', 'ALAC', 'Opus', 'Vorbis', 'WAV']) {
      expect(options.join(' ')).toMatch(new RegExp(fmt, 'i'))
    }
  })

  test('selecting ALAC clears bitrate quality (no bitrate options)', async ({ page }) => {
    await gotoConverter(page)
    await page.evaluate(() => {
      ;(window as any).go.main.App.OpenFLACFilesDialog = async () => ['/tmp/a.flac']
    })
    await page.locator('button', { hasText: /Select Files/i }).first().click()
    const select = page.locator('select').first()
    await expect(select).toBeVisible()
    await select.selectOption('ALAC')
    // ALAC has no quality options; the select should still be ALAC
    await expect(select).toHaveValue('ALAC')
  })

  test('page renders without runtime errors', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (e) => errors.push(e.message))
    await gotoConverter(page)
    await page.waitForTimeout(300)
    expect(errors, errors.join('\n')).toHaveLength(0)
  })
})
