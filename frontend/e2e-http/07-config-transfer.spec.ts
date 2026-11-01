import fs from 'node:fs'
import { test, expect, request, type APIRequestContext } from '@playwright/test'
import { openView, readServerInfo, setFolder } from './helpers'
import { EXT_PORT, TOKEN, TOKEN_URL } from './servers'
// @ts-expect-error plain .mjs helper without type declarations
import { makeTempRoot, removeTempRoot, startServer } from './ext-server.mjs'

const SECRETS = {
  tidalClientId: 'tid-secret',
  tidalClientSecret: 'tsec-secret',
  soulseekUsername: 'su-secret',
  soulseekPassword: 'sp-secret',
  qobuzAppId: 'qid-secret',
  qobuzAppSecret: 'qsec-secret',
  qobuzAuthToken: 'qtok-secret',
}

async function saveSecrets(api: APIRequestContext, extra: Record<string, unknown> = {}) {
  const cfg = await (await api.get('/api/config')).json()
  const res = await api.post('/api/config', { data: { ...cfg, ...SECRETS, ...extra } })
  expect(res.ok(), await res.text()).toBeTruthy()
}

test.describe.configure({ mode: 'serial' })

test.describe('Config export and import in the browser', () => {
  test.beforeEach(async ({ request }) => {
    await setFolder(request, readServerInfo('open').downloadDir)
  })

  test('the Export button downloads a valid file with every secret blanked', async ({ page, request }) => {
    await saveSecrets(request)
    await page.goto('/')
    await openView(page, 'Settings')
    const download = page.waitForEvent('download')
    await page.getByRole('button', { name: 'Export Config' }).click()
    const file = await download
    expect(file.suggestedFilename()).toBe('flacidal-config.json')

    const text = fs.readFileSync((await file.path())!, 'utf8')
    const parsed = JSON.parse(text)
    expect(parsed.schemaVersion).toBe(1)
    for (const key of Object.keys(SECRETS)) expect(parsed.config[key] ?? '').toBe('')
    for (const value of Object.values(SECRETS)) expect(text).not.toContain(value)
    await expect(page.locator('.toast').first()).toContainText('Config exported')
  })

  test('the server token never appears in an export', async ({ playwright }) => {
    const api = await playwright.request.newContext({ baseURL: TOKEN_URL, extraHTTPHeaders: { Authorization: `Bearer ${TOKEN}` } })
    const res = await api.get('/api/config/export')
    expect(res.ok()).toBe(true)
    expect(await res.text()).not.toContain(TOKEN)
    await api.dispose()
  })

  test('importing an export applies it and keeps the secrets already saved', async ({ page, request }) => {
    await saveSecrets(request, { language: 'en' })
    const exported = await (await request.get('/api/config/export')).text()
    const file = JSON.parse(exported)
    file.config.language = 'de'

    await page.goto('/')
    await openView(page, 'Settings')
    const chooser = page.waitForEvent('filechooser')
    await page.getByRole('button', { name: 'Import Config' }).click()
    await (await chooser).setFiles({ name: 'flacidal-config.json', mimeType: 'application/json', buffer: Buffer.from(JSON.stringify(file)) })
    await expect(page.locator('.toast').first()).toBeVisible() // worded in German: the import switched the language

    const cfg = await (await request.get('/api/config')).json()
    expect(cfg.language).toBe('de')
    for (const [key, value] of Object.entries(SECRETS)) expect(cfg[key]).toBe(value)

    await request.post('/api/config', { data: { ...cfg, language: 'en' } })
  })

  test('a file that is not a FLACidal export gives an error toast and changes nothing', async ({ page, request }) => {
    const before = await (await request.get('/api/config')).json()
    await page.goto('/')
    await openView(page, 'Settings')
    const chooser = page.waitForEvent('filechooser')
    await page.getByRole('button', { name: 'Import Config' }).click()
    await (await chooser).setFiles({ name: 'bad.json', mimeType: 'application/json', buffer: Buffer.from('{"hello":"world"}') })
    await expect(page.locator('.toast').first()).toContainText('Error importing config')
    expect(await (await request.get('/api/config')).json()).toEqual(before)
  })

  test('the import route answers 400 for garbage and 401 without a token', async ({ request, playwright }) => {
    const bad = await request.post('/api/config/import', { data: 'LEAKME {not json', headers: { 'Content-Type': 'application/json' } })
    expect(bad.status()).toBe(400)
    expect(await bad.text()).not.toContain('LEAKME')

    const anon = await playwright.request.newContext({ baseURL: TOKEN_URL })
    expect((await anon.post('/api/config/import', { data: '{}' })).status()).toBe(401)
    expect((await anon.get('/api/config/export')).status()).toBe(401)
    await anon.dispose()
  })
})

test('export, import and a server restart round trip', async () => {
  const port = EXT_PORT + 2
  const url = `http://127.0.0.1:${port}`
  const root = makeTempRoot()
  let server = await startServer({ root, token: TOKEN, port, certPath: '' })
  let api = await request.newContext({ baseURL: url, extraHTTPHeaders: { Authorization: `Bearer ${TOKEN}` } })
  try {
    await saveSecrets(api, { language: 'fr' })
    const exported = JSON.parse(await (await api.get('/api/config/export')).text())
    exported.config.language = 'de'
    exported.config.qobuzAppId = 'qid-from-file'

    const res = await api.post('/api/config/import', { data: exported })
    expect(res.ok(), await res.text()).toBeTruthy()

    await api.dispose()
    await server.stop()
    server = await startServer({ root, token: TOKEN, port, certPath: '' })
    api = await request.newContext({ baseURL: url, extraHTTPHeaders: { Authorization: `Bearer ${TOKEN}` } })

    const cfg = await (await api.get('/api/config')).json()
    expect(cfg.language).toBe('de')
    expect(cfg.qobuzAppId).toBe('qid-from-file')
    expect(cfg.qobuzAppSecret).toBe(SECRETS.qobuzAppSecret)
    expect(cfg.tidalClientSecret).toBe(SECRETS.tidalClientSecret)
  } finally {
    await api.dispose()
    await server.stop()
    removeTempRoot(root)
  }
})
