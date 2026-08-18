import { test, expect } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

/**
 * As with the other pages, App.svelte's `{#key activePage}` + transition:fade
 * leaves two DOM copies around during the 150ms fade — stick to `.first()` and
 * give the transition a moment to finish.
 */
test.describe('History screen', () => {
  test('shows the empty state when history has no records', async ({ page }) => {
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

  test('shows table headers once records are loaded', async ({ page }) => {
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
