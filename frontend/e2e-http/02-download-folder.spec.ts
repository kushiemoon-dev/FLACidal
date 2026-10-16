import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { test, expect, type Page } from '@playwright/test'
import { openView, readServerInfo, setFolder } from './helpers'

const info = () => readServerInfo('open')

async function typeFolder(page: Page, value: string) {
  await page.goto('/')
  await openView(page, 'Settings')
  const input = page.locator('#download-folder')
  await expect(input).toBeVisible()
  await input.fill(value)
  await input.blur()
}

test.describe('Download folder on the real server', () => {
  let tmp: string

  test.beforeEach(async ({ request }) => {
    tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'flacidal-e2e-folder-'))
    await setFolder(request, info().downloadDir)
  })

  test.afterEach(() => {
    fs.chmodSync(tmp, 0o755)
    fs.rmSync(tmp, { recursive: true, force: true })
  })

  test('a valid folder is saved and read back after a reload', async ({ page }) => {
    const target = path.join(tmp, 'music')
    fs.mkdirSync(target)

    const saved = page.waitForResponse((r) => r.url().endsWith('/api/folder') && r.request().method() === 'POST')
    await typeFolder(page, target)
    expect((await saved).ok()).toBe(true)
    await expect(page.locator('[role="alert"] .toast')).toHaveCount(0)

    await page.reload()
    await openView(page, 'Settings')
    await expect(page.locator('#download-folder')).toHaveValue(target)
  })

  test('a missing folder is refused and the old value is kept', async ({ page }) => {
    const missing = path.join(tmp, 'nope')
    await typeFolder(page, missing)

    await expect(page.locator('.toast').first()).toContainText('does not exist')
    await expect(page.locator('#download-folder')).toHaveValue(info().downloadDir)
  })

  test('a file instead of a folder is refused', async ({ page }) => {
    const file = path.join(tmp, 'plain.txt')
    fs.writeFileSync(file, 'x')
    await typeFolder(page, file)

    await expect(page.locator('.toast').first()).toContainText('is a file, not a folder')
    await expect(page.locator('#download-folder')).toHaveValue(info().downloadDir)
  })

  test('a folder we cannot write in is refused', async ({ page }) => {
    test.skip(process.getuid?.() === 0, 'root ignores directory permissions')
    const ro = path.join(tmp, 'readonly')
    fs.mkdirSync(ro)
    fs.chmodSync(ro, 0o555)

    await typeFolder(page, ro)

    await expect(page.locator('.toast').first()).toContainText('cannot write')
    await expect(page.locator('#download-folder')).toHaveValue(info().downloadDir)
    fs.chmodSync(ro, 0o755)
  })
})
