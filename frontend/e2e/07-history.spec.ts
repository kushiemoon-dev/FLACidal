import { test, expect } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

/**
 * App.svelte's `{#key activePage}` + transition:fade keeps two DOM copies during
 * the 150ms fade — always use `.first()` and wait for the transition.
 */
test.describe('History page', () => {
  test('loads with empty history shows empty state', async ({ page }) => {
    await injectWailsMocks(page, {
      GetDownloadHistoryFiltered: { records: [], total: 0 },
    })
    await page.goto('/')
    await page.locator('.sidebar button[title="History"]').click()
    await page.waitForTimeout(250)

    await expect(page.locator('h1, h2').filter({ hasText: /History/i }).first()).toBeVisible()
    await expect(
      page.locator('text=/No download history|Your downloaded tracks/i').first(),
    ).toBeVisible()
  })

  test('loads with records → table headers visible', async ({ page }) => {
    await injectWailsMocks(page, {
      GetDownloadHistoryFiltered: {
        records: [
          {
            id: 1,
            tidalContentId: 'tid-album-1',
            tidalContentName: 'Test Album',
            contentType: 'album',
            tracksDownloaded: 10,
            tracksTotal: 12,
            tracksFailed: 0,
            lastDownloadAt: new Date().toISOString(),
            firstDownloadAt: new Date().toISOString(),
          },
        ],
        total: 1,
      },
    })
    await page.goto('/')
    await page.locator('.sidebar button[title="History"]').click()
    await page.waitForTimeout(250)

    await expect(page.locator('.history-table').first()).toBeVisible()
    for (const header of ['Name', 'Type', 'Tracks', 'Last Download', 'Actions']) {
      await expect(
        page.locator('.table-header .th', { hasText: header }).first(),
      ).toBeVisible()
    }
    await expect(page.locator('.content-name', { hasText: 'Test Album' }).first()).toBeVisible()
  })
})
