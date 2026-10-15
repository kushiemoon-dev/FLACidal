import { test, expect, type Page } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

const FILES = [1, 2, 3].map((n) => ({
  path: `/music/track-${n}.flac`,
  name: `track-${n}.flac`,
  size: 1024 * 1024 * 20 * n,
  modTime: `2026-10-0${n}T10:00:00Z`,
  title: `Track ${n}`,
  artist: `Artist ${n}`,
  album: `Album ${n}`,
}))

async function openFiles(page: Page, overrides = {}) {
  await injectWailsMocks(page, { ListDownloadedFiles: FILES, ...overrides })
  await page.goto('/')
  await page.locator('.sidebar button[title="Files"]').click()
}

test.describe('Keyboard and screen reader behaviour', () => {
  test('row checkbox toggles with Space and the header one turns indeterminate', async ({ page }) => {
    await openFiles(page)
    await expect(page.locator('.table-row')).toHaveCount(3)

    const row = page.getByRole('checkbox', { name: 'Select Track 1' })
    await row.focus()
    await page.keyboard.press('Space')
    await expect(row).toBeChecked()
    await expect(page.locator('.selection-count')).toContainText('1 selected')

    const all = page.getByRole('checkbox', { name: 'Select all files' })
    expect(await all.evaluate((el: HTMLInputElement) => el.indeterminate)).toBe(true)
  })

  test('the focused checkbox has a visible indicator', async ({ page }) => {
    await openFiles(page)
    await expect(page.locator('.table-row')).toHaveCount(3)
    await page.getByRole('checkbox', { name: 'Select all files' }).focus()
    await page.keyboard.press('Tab')
    const outline = await page.locator('.table-header .custom-checkbox').evaluate((el) => getComputedStyle(el).outlineStyle)
    await page.getByRole('checkbox', { name: 'Select all files' }).focus()
    const focused = await page.locator('.table-header .custom-checkbox').evaluate((el) => getComputedStyle(el).outlineStyle)
    expect(outline).toBe('none')
    expect(focused).not.toBe('none')
  })

  test('focus outline is visible on a button after Tab', async ({ page }) => {
    await openFiles(page)
    await page.keyboard.press('Tab')
    const style = await page.evaluate(() => getComputedStyle(document.activeElement as Element).outlineStyle)
    expect(style).not.toBe('none')
  })

  test('an error toast is announced as an alert', async ({ page }) => {
    await openFiles(page, { ListDownloadedFilesError: '[folder.missing] /gone' })
    await expect(page.locator('[role="alert"] .toast').first()).toContainText('/gone')
  })

  test('the delete dialog takes focus, closes on Escape and gives focus back', async ({ page }) => {
    await openFiles(page)
    const trigger = page.locator('.file-btn.danger').first()
    await trigger.focus()
    await page.keyboard.press('Enter')

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    expect(await dialog.evaluate((el) => el.contains(document.activeElement))).toBe(true)

    await page.keyboard.press('Escape')
    await expect(dialog).toHaveCount(0)
    await expect(trigger).toBeFocused()
  })

  test('the metadata modal is a named dialog that returns focus on Escape', async ({ page }) => {
    await openFiles(page)
    const trigger = page.locator('.file-btn.info').first()
    await trigger.focus()
    await page.keyboard.press('Enter')

    const dialog = page.getByRole('dialog', { name: /metadata/i })
    await expect(dialog).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(dialog).toHaveCount(0)
    await expect(trigger).toBeFocused()
  })
})
