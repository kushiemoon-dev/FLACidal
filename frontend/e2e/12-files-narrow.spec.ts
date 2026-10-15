import { test, expect, type Page } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

const FILES = [1, 2, 3, 4].map((n) => ({
  path: `/music/a-rather-long-folder-name/track-${n}.flac`,
  name: `a-rather-long-file-name-for-track-${n}.flac`,
  size: 1024 * 1024 * 20 * n,
  modTime: `2026-10-0${n}T10:00:00Z`,
  title: `A fairly long track title number ${n}`,
  artist: `Artist ${n}`,
  album: `Album with a long name ${n}`,
}))

async function openFiles(page: Page, converter = true) {
  await injectWailsMocks(page, { ListDownloadedFiles: FILES, IsConverterAvailable: converter })
  await page.goto('/')
  await page.locator('.sidebar button[title="Files"]').click()
  await expect(page.locator('.table-row')).toHaveCount(FILES.length)
}

async function expectNoOverflow(page: Page) {
  const overflow = await page.evaluate(() => ({
    doc: document.documentElement.scrollWidth - window.innerWidth,
    table: (document.querySelector('.files-table') as HTMLElement).scrollWidth - (document.querySelector('.files-table') as HTMLElement).clientWidth,
  }))
  expect(overflow.doc).toBeLessThanOrEqual(0)
  expect(overflow.table).toBeLessThanOrEqual(0)
}

async function expectActionsFit(page: Page) {
  const buttons = page.locator('.header-actions .action-btn')
  const count = await buttons.count()
  expect(count).toBeGreaterThan(0)
  for (let i = 0; i < count; i++) {
    const box = await buttons.nth(i).boundingBox()
    expect(box).not.toBeNull()
    expect(box!.x).toBeGreaterThanOrEqual(0)
    expect(box!.x + box!.width).toBeLessThanOrEqual(390)
    expect(box!.height).toBeGreaterThanOrEqual(44)
  }
}

test.describe('Files at 390px', () => {
  test.use({ viewport: { width: 390, height: 844 } })

  for (const converter of [true, false]) {
    test(`actions and table fit without selection (converter ${converter})`, async ({ page }) => {
      await openFiles(page, converter)
      await expectNoOverflow(page)
      await expectActionsFit(page)
    })

    test(`actions fit with a selection (converter ${converter})`, async ({ page }) => {
      await openFiles(page, converter)
      await page.locator('.table-row .checkbox-col').first().click()
      await expect(page.locator('.selection-count')).toBeVisible()
      await expectNoOverflow(page)
      await expectActionsFit(page)
    })
  }

  test('sort buttons and every row cell stay visible', async ({ page }) => {
    await openFiles(page)
    await expect(page.locator('.th.sortable')).toHaveCount(3)
    for (const sort of await page.locator('.th.sortable').all()) await expect(sort).toBeVisible()

    const row = page.locator('.table-row').first()
    await expect(row.locator('.file-name')).toBeVisible()
    for (const cell of ['.artist', '.album', '.size', '.date']) await expect(row.locator(cell)).toBeVisible()
    await expect(row.locator('.file-btn')).toHaveCount(3)
  })
})

test.describe('Files at 1024px', () => {
  test.use({ viewport: { width: 1024, height: 768 } })

  test('keeps the desktop layout', async ({ page }) => {
    await openFiles(page)
    const columns = await page.locator('.table-header').evaluate((el) => getComputedStyle(el).gridTemplateColumns.split(' ').length)
    expect(columns).toBe(7)
    const direction = await page.locator('.files-header').evaluate((el) => getComputedStyle(el).flexDirection)
    expect(direction).toBe('row')
  })
})
