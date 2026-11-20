import { test, expect, type Page } from '@playwright/test'
import { injectWailsMocks, type WinAny } from './mocks/wails'
import { expectNoBlockingViolations, setTheme } from './axe'

const ALBUMS = [
  { id: 1, title: 'Geogaddi', artist: 'Boards of Canada', releaseDate: '2002-02-18', trackCount: 23, coverUrl: '', label: 'Warp Records' },
  { id: 2, title: 'Dedicated', artist: 'Carly Rae Jepsen', releaseDate: '2019-05-17', trackCount: 15, coverUrl: '', label: 'Interscope' },
  { id: 3, title: 'Mystery', artist: 'Nobody', releaseDate: '2020-01-01', trackCount: 1, coverUrl: '' },
]
const TRACKS = [
  { id: 11, title: 'Roygbiv', artist: 'Boards of Canada', artists: 'Boards of Canada', album: 'Music Has the Right to Children', coverUrl: '', duration: 151 },
]

async function openSearch(page: Page, tab: 'Tracks' | 'Albums') {
  await injectWailsMocks(page, { SearchTidal: TRACKS, SearchTidalAlbums: ALBUMS, GetDownloadFolder: '/music' })
  await page.goto('/')
  await page.locator('.sidebar button[title="Search"]').click()
  await page.waitForTimeout(250)
  await page.locator('.search-tabs .tab', { hasText: tab }).click()
  await page.locator('.search-input').fill('boards')
  await page.locator('.search-btn').click()
}

test.describe('Search albums label filter', () => {
  test('keeps only the matching label and shows labels on the cards', async ({ page }) => {
    await openSearch(page, 'Albums')
    await expect(page.locator('.album-card')).toHaveCount(3)
    await expect(page.locator('.album-card-label', { hasText: 'Warp Records' })).toBeVisible()

    await page.locator('.label-filter-input').fill('warp')
    await expect(page.locator('.album-card')).toHaveCount(1)
    await expect(page.locator('.album-card-title')).toHaveText('Geogaddi')
    await expect(page.locator('.label-unknown-count')).toContainText('1')
    await expect(page.locator('#label-filter-hint')).toContainText('does not start a new search')
  })

  for (const theme of ['dark', 'light'] as const) {
    test(`albums tab has no blocking accessibility violations (${theme})`, async ({ page }) => {
      await openSearch(page, 'Albums')
      await expect(page.locator('.album-card')).toHaveCount(3)
      await page.locator('.label-filter-input').fill('a')
      await setTheme(page, theme)
      await expectNoBlockingViolations(page, `search albums ${theme}`)
    })
  }
})

test.describe('Search Atmos buttons', () => {
  test('track Atmos queues ATMOS, the plain button queues without quality', async ({ page }) => {
    await openSearch(page, 'Tracks')
    await page.locator('.track-atmos-btn').click()
    await page.locator('.track-download-btn').click()
    const calls = await page.evaluate(() => (window as unknown as WinAny).__calls)
    const singles = calls?.filter((c) => c.name === 'QueueSingleDownload') ?? []
    expect(singles.map((c) => c.args[4])).toEqual(['ATMOS', ''])
    expect(calls?.some((c) => c.name === 'SetDownloadOptions')).toBeFalsy()
  })

  test('album Atmos queues the album with ATMOS', async ({ page }) => {
    await openSearch(page, 'Albums')
    await page.locator('.album-atmos-btn').first().click()
    const calls = await page.evaluate(() => (window as unknown as WinAny).__calls)
    const album = calls?.find((c) => c.name === 'QueueArtistAlbum')
    expect(album?.args[3]).toBe('ATMOS')
  })

  for (const theme of ['dark', 'light'] as const) {
    test(`tracks tab has no blocking accessibility violations (${theme})`, async ({ page }) => {
      await openSearch(page, 'Tracks')
      await expect(page.locator('.track-atmos-btn')).toBeVisible()
      await setTheme(page, theme)
      await expectNoBlockingViolations(page, `search tracks ${theme}`)
    })
  }
})

const FILES = [
  { path: '/music/a.flac', name: 'a.flac', size: 1e7, modTime: '2026-10-01T10:00:00Z', title: 'Flac Song', artist: 'X', album: 'Y', format: 'flac' },
  { path: '/music/b.m4a', name: 'b.m4a', size: 2e7, modTime: '2026-10-02T10:00:00Z', title: 'Atmos Song', artist: 'X', album: 'Y', format: 'm4a', codec: 'eac3', quality: 'Dolby Atmos' },
  { path: '/music/c.m4a', name: 'c.m4a', size: 3e6, modTime: '2026-10-03T10:00:00Z', title: 'Aac Song', artist: 'X', album: 'Y', format: 'm4a', codec: 'aac' },
]

test.describe('Files with .m4a', () => {
  async function openFiles(page: Page) {
    await injectWailsMocks(page, { ListDownloadedFiles: FILES, GetDownloadFolder: '/music' })
    await page.goto('/')
    await page.locator('.sidebar button[title="Files"]').click()
    await page.waitForTimeout(250)
  }

  test('shows an Atmos badge and an M4A badge, none on FLAC', async ({ page }) => {
    await openFiles(page)
    await expect(page.locator('.format-badge.atmos')).toHaveCount(1)
    await expect(page.locator('.format-badge.m4a')).toHaveCount(1)
    await expect(page.locator('.table-row', { hasText: 'a.flac' }).locator('.format-badge')).toHaveCount(0)
  })

  for (const theme of ['dark', 'light'] as const) {
    test(`has no blocking accessibility violations (${theme})`, async ({ page }) => {
      await openFiles(page)
      await expect(page.locator('.format-badge.atmos')).toBeVisible()
      await setTheme(page, theme)
      await expectNoBlockingViolations(page, `files m4a ${theme}`)
    })
  }
})
