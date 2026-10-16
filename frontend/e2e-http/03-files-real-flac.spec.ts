import fs from 'node:fs'
import path from 'node:path'
import { test, expect } from '@playwright/test'
import { openView, seedFlac, setFolder } from './helpers'

const NAMES = ['alpha.flac', 'beta.flac', 'gamma.flac']

test.describe('Files on real FLAC', () => {
  let dir: string

  test.beforeEach(async ({ request, page }) => {
    dir = seedFlac(NAMES)
    await setFolder(request, dir)
    await page.goto('/')
    await openView(page, 'Files')
  })

  test.afterEach(() => {
    fs.rmSync(dir, { recursive: true, force: true })
  })

  test('lists the three files with title, artist and album', async ({ page }) => {
    const rows = page.locator('.table-row')
    await expect(rows).toHaveCount(3)
    for (const n of ['Alpha', 'Beta', 'Gamma']) {
      const row = rows.filter({ hasText: `${n} Song` })
      await expect(row).toContainText(`${n} Artist`)
      await expect(row).toContainText(`${n} Album`)
    }
  })

  test('opens the metadata modal for a file', async ({ page }) => {
    await page.locator('.table-row').filter({ hasText: 'Alpha Song' }).locator('.file-btn.info').click()
    const dialog = page.locator('[role="dialog"]')
    await expect(dialog).toContainText('Alpha Song')
    await expect(dialog).toContainText('Alpha Artist')
  })

  test('deletes a file from the list and from the disk', async ({ page }) => {
    const target = path.join(dir, 'beta.flac')
    expect(fs.existsSync(target)).toBe(true)

    await page.locator('.table-row').filter({ hasText: 'Beta Song' }).locator('.file-btn.danger').click()
    await page.locator('[role="dialog"] .btn-confirm').click()

    await expect(page.locator('.table-row')).toHaveCount(2)
    await expect(page.locator('.table-row').filter({ hasText: 'Beta Song' })).toHaveCount(0)
    expect(fs.existsSync(target)).toBe(false)
  })

  test('renames a file on the disk', async ({ page }) => {
    await page.locator('.table-row').filter({ hasText: 'Gamma Song' }).getByRole('checkbox').check()
    await page.locator('.action-btn.primary', { hasText: 'Rename' }).click()

    const dialog = page.locator('[role="dialog"]')
    const newName = dialog.locator('.preview-new span').first()
    await expect(newName).toBeVisible()
    const target = (await newName.textContent())?.trim() ?? ''
    expect(target).not.toBe('')
    expect(target).not.toBe('gamma.flac')

    await dialog.getByRole('button', { name: /Rename 1 File/ }).click()

    await expect.poll(() => fs.existsSync(path.join(dir, 'gamma.flac'))).toBe(false)
    expect(fs.existsSync(path.join(dir, target))).toBe(true)
  })
})
