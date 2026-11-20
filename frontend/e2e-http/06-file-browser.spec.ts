import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { createRequire } from 'node:module'
import { test, expect } from '@playwright/test'
import { openTool, openView, pickCurrentFolder, pickFiles, pickFolderAt, seedFlac, setFolder } from './helpers'

const axePath = createRequire(import.meta.url).resolve('axe-core/axe.min.js')
const NAMES = ['alpha.flac', 'beta.flac', 'gamma.flac']

test.describe('Server file browser in the tools', () => {
  let dir: string

  test.beforeEach(async ({ request }) => {
    dir = seedFlac(NAMES)
    await setFolder(request, dir)
  })

  test.afterEach(() => {
    fs.rmSync(dir, { recursive: true, force: true })
  })

  test('Audio Quality Analyzer: files, then a whole folder', async ({ page }) => {
    await page.goto('/')
    await openTool(page, 'Audio Quality Analyzer')
    await page.getByRole('button', { name: 'Select Files' }).click()
    await pickFiles(page, 'alpha.flac', 'beta.flac')
    await expect(page.locator('.table-row')).toHaveCount(2)

    await page.getByRole('button', { name: 'Analyze More' }).click()
    await page.getByRole('button', { name: 'Select Folder' }).click()
    await pickCurrentFolder(page)
    await expect(page.locator('.table-row')).toHaveCount(3)
  })

  test('Audio Converter: files and an output folder', async ({ page }) => {
    await page.goto('/')
    await openTool(page, 'Audio Converter')
    await page.getByRole('button', { name: 'Select Files' }).click()
    await pickFiles(page, 'alpha.flac', 'gamma.flac')
    await expect(page.getByText('alpha.flac')).toBeVisible()
    await expect(page.getByText('gamma.flac')).toBeVisible()

    await page.locator('.btn.btn-accent', { hasText: /Browse/i }).first().click()
    await pickCurrentFolder(page)
    await expect(page.getByRole('textbox', { name: 'Output Folder' })).toHaveValue(dir)
  })

  test('Audio Resampler: files and an output folder', async ({ page }) => {
    await page.goto('/')
    await openTool(page, 'Audio Resampler')
    await page.getByRole('button', { name: 'Select Files' }).click()
    await pickFiles(page, 'beta.flac')
    await expect(page.getByText('1 file selected')).toBeVisible()

    await expect(page.getByText('Not selected')).toBeVisible()
    await page.getByRole('button', { name: 'Browse' }).first().click()
    await pickCurrentFolder(page)
    await expect(page.getByText('Not selected')).toHaveCount(0)
  })

  test('Lyrics Manager: files', async ({ page }) => {
    await page.goto('/')
    await openTool(page, 'Lyrics Manager')
    await page.getByRole('button', { name: 'Select Files' }).click()
    await pickFiles(page, 'alpha.flac', 'beta.flac', 'gamma.flac')
    for (const n of NAMES) await expect(page.getByText(n)).toBeVisible()
  })

  test('File Manager: a folder fills the list', async ({ page }) => {
    await page.goto('/')
    await openTool(page, 'File Manager')
    // The File Manager picks its folder anywhere on the server, not only inside the allowed ones.
    await page.locator('.folder-bar .btn-accent').click()
    await pickFolderAt(page, dir)
    await expect(page.locator('.folder-bar input')).toHaveValue(dir)
    await expect(page.getByText('3 files')).toBeVisible()
  })

  test('works from the keyboard alone and gives the focus back', async ({ page }) => {
    await page.goto('/')
    await openTool(page, 'Lyrics Manager')
    const opener = page.getByRole('button', { name: 'Select Files' })
    await opener.focus()
    await page.keyboard.press('Enter')

    const dialog = page.locator('[role="dialog"]')
    await expect(dialog).toBeVisible()
    await expect(dialog.getByLabel('alpha.flac')).toBeVisible()
    // Focus stays inside the dialog however many times Tab is pressed.
    for (let i = 0; i < 12; i++) {
      await page.keyboard.press('Tab')
      expect(await dialog.evaluate((d) => d.contains(document.activeElement))).toBe(true)
    }
    await dialog.getByLabel('beta.flac').focus()
    await page.keyboard.press('Space')
    await dialog.getByRole('button', { name: 'Select', exact: true }).focus()
    await page.keyboard.press('Enter')
    await expect(dialog).toHaveCount(0)
    await expect(page.getByText('beta.flac')).toBeVisible()

    await opener.or(page.getByRole('button', { name: 'Add More' })).first().focus()
    await page.keyboard.press('Enter')
    await expect(dialog).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(dialog).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Add More' })).toBeFocused()
  })

  for (const theme of ['dark', 'light'] as const) {
    test(`passes axe in the ${theme} theme`, async ({ page }) => {
      await page.goto('/')
      await page.evaluate((t) => document.documentElement.setAttribute('data-theme', t), theme)
      await openTool(page, 'Lyrics Manager')
      await page.getByRole('button', { name: 'Select Files' }).click()
      await expect(page.locator('[role="dialog"]').getByLabel('alpha.flac')).toBeVisible()
      await page.addScriptTag({ path: axePath })
      const violations = await page.evaluate(async () => {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const r = await (window as any).axe.run(document.querySelector('[role="dialog"]'))
        return r.violations.map((v: { id: string; nodes: { target: unknown[] }[] }) => `${v.id}: ${v.nodes.map((n) => n.target.join(' ')).join(' | ')}`)
      })
      expect(violations).toEqual([])
    })
  }
})

test.describe('Browse routes on the real server', () => {
  let dir: string
  let outside: string

  test.beforeEach(async ({ request }) => {
    dir = seedFlac(NAMES)
    outside = fs.mkdtempSync(path.join(os.tmpdir(), 'flacidal-e2e-outside-'))
    fs.symlinkSync(outside, path.join(dir, 'escape'))
    await setFolder(request, dir)
  })

  test.afterEach(() => {
    fs.rmSync(dir, { recursive: true, force: true })
    fs.rmSync(outside, { recursive: true, force: true })
  })

  test('tools scope refuses paths outside the allowed folders', async ({ request }) => {
    const get = (p: string) => request.get(`/api/browse?scope=tools&path=${encodeURIComponent(p)}`)
    expect((await get('/etc')).status()).toBe(403)
    expect((await get(`${dir}/../..`)).status()).toBe(403)
    expect((await get(path.join(dir, 'escape'))).status()).toBe(403)
  })

  test('tools scope hides outgoing symlinks and lists only FLAC files when asked', async ({ request }) => {
    const res = await request.get(`/api/browse?scope=tools&files=1&path=${encodeURIComponent(dir)}`)
    expect(res.ok()).toBe(true)
    const names = (await res.json()).entries.map((e: { name: string }) => e.name)
    expect(names).toEqual(NAMES)
  })

  test('folders scope lists folders anywhere and never files', async ({ request }) => {
    fs.mkdirSync(path.join(outside, 'sub'))
    fs.writeFileSync(path.join(outside, 'x.flac'), 'x')
    const res = await request.get(`/api/browse?scope=folders&files=1&path=${encodeURIComponent(outside)}`)
    expect(res.ok()).toBe(true)
    const names = (await res.json()).entries.map((e: { name: string }) => e.name)
    expect(names).toEqual(['sub'])
  })

  test('/api/browse/flac is recursive and confined', async ({ request }) => {
    fs.mkdirSync(path.join(dir, 'deep'))
    fs.copyFileSync(path.join(dir, 'alpha.flac'), path.join(dir, 'deep', 'delta.flac'))
    const ok = await request.get(`/api/browse/flac?path=${encodeURIComponent(dir)}`)
    expect((await ok.json()).paths).toHaveLength(4)
    expect((await request.get(`/api/browse/flac?path=${encodeURIComponent(outside)}`)).status()).toBe(403)
  })

  test('browse needs the token on the protected server', async ({ playwright }) => {
    const { TOKEN_URL } = await import('./servers')
    const anon = await playwright.request.newContext({ baseURL: TOKEN_URL })
    expect((await anon.get('/api/browse?scope=folders&path=/tmp')).status()).toBe(401)
    expect((await anon.get('/api/browse/flac?path=/tmp')).status()).toBe(401)
    await anon.dispose()
  })
})

test.describe('Choosing a download folder anywhere', () => {
  let tmp: string
  let start: string

  test.beforeEach(async ({ request }) => {
    tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'flacidal-e2e-pick-'))
    start = seedFlac(NAMES)
    await setFolder(request, start)
  })

  test.afterEach(() => {
    fs.rmSync(tmp, { recursive: true, force: true })
    fs.rmSync(start, { recursive: true, force: true })
  })

  test('a folder outside the old one can be chosen and saved, with no file in the list', async ({ page, request }) => {
    const target = path.join(tmp, 'new-music')
    fs.mkdirSync(target)
    fs.writeFileSync(path.join(target, 'song.flac'), 'x')

    await page.goto('/')
    await openView(page, 'Settings')
    await page.locator('.browse-btn').first().click()
    const dialog = page.locator('[role="dialog"]')
    await expect(dialog).toBeVisible()
    await dialog.getByLabel('Folder path').fill(target)
    await dialog.getByLabel('Folder path').press('Enter')
    await expect(dialog.getByLabel('Folder path')).toHaveValue(target)
    await expect(dialog.locator('input[type="checkbox"]')).toHaveCount(0)
    await expect(dialog.getByText('song.flac')).toHaveCount(0)
    await dialog.getByRole('button', { name: 'Choose this folder' }).click()

    await expect(page.locator('#download-folder')).toHaveValue(target)
    const cfg = await (await request.get('/api/config')).json()
    expect(cfg.downloadFolder).toBe(target)
  })

  test('the root folder is refused when saving', async ({ page, request }) => {
    await page.goto('/')
    await openView(page, 'Settings')
    await page.locator('.browse-btn').first().click()
    const dialog = page.locator('[role="dialog"]')
    await dialog.getByLabel('Folder path').fill('/')
    await dialog.getByLabel('Folder path').press('Enter')
    await expect(dialog.getByLabel('Folder path')).toHaveValue('/')
    await dialog.getByRole('button', { name: 'Choose this folder' }).click()

    await expect(page.locator('.toast').first()).toBeVisible()
    const cfg = await (await request.get('/api/config')).json()
    expect(cfg.downloadFolder).toBe(start)
  })
})
