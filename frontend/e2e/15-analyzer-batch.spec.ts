import { test, expect, type Page } from '@playwright/test'
import { injectWailsMocks, type WinAny } from './mocks/wails'
import { expectNoBlockingViolations, setTheme } from './axe'

const row = (name: string) => ({
  filePath: `/music/${name}`, fileName: name, verdict: 'lossless', verdictLabel: 'True Lossless',
  isTrueLossless: true, confidence: 95, spectrumCutoff: 22000, expectedCutoff: 22050,
  details: '', sampleRate: 44100, bitDepth: 16, bpm: 0, musicalKey: '',
})
const errorRow = (name: string) => ({
  ...row(name), verdict: 'error', verdictLabel: 'Error', isTrueLossless: false, confidence: 0,
  details: 'cannot decode', sampleRate: 0, bitDepth: 0,
})

const PATHS = ['/music/a.flac', '/music/bad.flac', '/music/c.flac']
const RESULTS = [row('a.flac'), errorRow('bad.flac'), row('c.flac')]

async function startBatch(page: Page, delayMs: number) {
  await injectWailsMocks(page, { OpenFLACFilesDialog: PATHS, AnalyzeBatch: { results: RESULTS, delayMs } })
  await page.goto('/')
  await page.locator('.sidebar button[title="Tools"]').click()
  await page.locator('.flyout-item', { hasText: 'Audio Quality Analyzer' }).click()
  await page.waitForTimeout(250)
  await page.getByRole('button', { name: 'Select Files' }).first().click()
}

test.describe('Audio Quality Analyzer batch', () => {
  test('shows progress, then lists the failing file apart from the results', async ({ page }) => {
    await startBatch(page, 300)
    await expect.poll(() => page.locator('.progress-text').first().textContent(), { intervals: [50] }).toContain('2 of 3')
    await expect(page.locator('progress.analyze-progress').first()).toBeVisible()

    await expect(page.locator('.errors-panel').first()).toContainText('bad.flac')
    await expect(page.locator('.errors-panel').first()).toContainText('cannot decode')
    await expect(page.locator('progress.analyze-progress')).toHaveCount(0, { timeout: 5000 })
    await expect(page.locator('.results-table .table-row')).toHaveCount(2)
  })

  test('Cancel keeps the results received so far', async ({ page }) => {
    await startBatch(page, 800)
    await expect.poll(() => page.locator('.progress-text').first().textContent(), { intervals: [50] }).toContain('1 of 3')
    await page.locator('.btn-cancel-analysis').first().click()

    await expect(page.locator('.cancelled-note').first()).toContainText('Analysis cancelled')
    await expect(page.locator('.results-table .table-row').first()).toBeVisible()
  })

  test('exports the report', async ({ page }) => {
    await startBatch(page, 0)
    await expect(page.locator('.btn-export-csv').first()).toBeEnabled()
    await page.locator('.btn-export-csv').first().click()
    const calls = await page.evaluate(() => (window as unknown as WinAny).__calls)
    const exp = calls?.find((c) => c.name === 'ExportAnalysisReport')
    expect(exp?.args[0]).toBe('csv')
  })

  for (const theme of ['dark', 'light'] as const) {
    test(`has no blocking accessibility violations (${theme})`, async ({ page }) => {
      await startBatch(page, 0)
      await expect(page.locator('.errors-panel').first()).toBeVisible()
      await setTheme(page, theme)
      await expectNoBlockingViolations(page, `analyzer ${theme}`)
    })
  }
})
