import { test, expect, request, type APIRequestContext } from '@playwright/test'
import { EXT_PORT, TOKEN } from './servers'
// @ts-expect-error plain .mjs helper without type declarations
import { buildZip, makeTempRoot, removeTempRoot, startServer, startZipServer } from './ext-server.mjs'

const API_PORT = EXT_PORT + 1
const API_URL = `http://127.0.0.1:${API_PORT}`
const SECRET = 'e2e-secret-value'

const manifest = (baseUrl: string) =>
  JSON.stringify({
    id: 'e2e-ext',
    name: 'E2E Extension',
    version: '1.0.0',
    author: 'e2e',
    category: 'download',
    capabilities: ['source'],
    permissions: ['network'],
    canDownload: true,
    downloadPriority: 1,
    sourceConfig: { name: 'e2e', displayName: 'E2E', urlPattern: 'e2e\\.example', baseUrl, trackEndpoint: '/t/{id}' },
    authFields: [{ key: 'X-Api-Key', label: 'API key', type: 'password', required: false }],
  })

interface Listed {
  manifest: { id: string }
  enabled: boolean
  authSet: Record<string, boolean>
}

test.describe.configure({ mode: 'serial' })

test('extension lifecycle on the real headless server', async () => {
  const root = makeTempRoot()
  const zipDir = makeTempRoot()
  const zipServer = await startZipServer(zipDir, EXT_PORT, buildZip([['extension.json', manifest(`https://127.0.0.1:${EXT_PORT}`)]]))
  let server = await startServer({ root, token: TOKEN, port: API_PORT, certPath: zipServer.certPath })
  let authed: APIRequestContext = await request.newContext({ baseURL: API_URL, extraHTTPHeaders: { Authorization: `Bearer ${TOKEN}` } })
  const anonymous = await request.newContext({ baseURL: API_URL })

  const list = async (): Promise<Listed[]> => {
    const res = await authed.get('/api/extensions')
    expect(res.ok(), await res.text()).toBeTruthy()
    return res.json()
  }
  const restart = async () => {
    await authed.dispose()
    await server.stop()
    server = await startServer({ root, token: TOKEN, port: API_PORT, certPath: zipServer.certPath })
    authed = await request.newContext({ baseURL: API_URL, extraHTTPHeaders: { Authorization: `Bearer ${TOKEN}` } })
  }

  try {
    // Without the token nothing is reachable.
    expect((await anonymous.get('/api/extensions')).status()).toBe(401)
    expect((await anonymous.post('/api/extensions/install', { data: { url: zipServer.url + '/ext.zip' } })).status()).toBe(401)

    // Plain http is refused with a stable code.
    const insecure = await authed.post('/api/extensions/install', { data: { url: 'http://example.com/e.zip' } })
    expect(insecure.status()).toBe(400)
    expect((await insecure.json()).error).toContain('[extension.insecure_url]')

    // Install: listed, disabled, no secret fields.
    const installed = await authed.post('/api/extensions/install', { data: { url: zipServer.url + '/ext.zip' } })
    expect(installed.ok(), await installed.text()).toBeTruthy()
    const afterInstall = await authed.get('/api/extensions')
    const text = await afterInstall.text()
    expect(text).not.toContain('authData')
    expect(text).not.toContain('"dir"')
    let ext = JSON.parse(text).find((e: Listed) => e.manifest.id === 'e2e-ext')
    expect(ext.enabled).toBe(false)

    // Enable and join the chain just before Soulseek.
    expect((await authed.post('/api/extensions/e2e-ext/enabled', { data: { enabled: true } })).ok()).toBeTruthy()
    const order: string[] = await (await authed.get('/api/sources/order')).json()
    expect(order.indexOf('e2e-ext')).toBe(order.indexOf('soulseek') - 1)

    // Credentials are accepted and never echoed back.
    const auth = await authed.post('/api/extensions/e2e-ext/auth', { data: { data: { 'X-Api-Key': SECRET } } })
    expect(auth.ok(), await auth.text()).toBeTruthy()
    expect(await auth.text()).not.toContain(SECRET)
    const withAuth = await (await authed.get('/api/extensions')).text()
    expect(withAuth).not.toContain(SECRET)
    ext = JSON.parse(withAuth).find((e: Listed) => e.manifest.id === 'e2e-ext')
    expect(ext.authSet['X-Api-Key']).toBe(true)

    // A disabled extension stays disabled after a restart.
    expect((await authed.post('/api/extensions/e2e-ext/enabled', { data: { enabled: false } })).ok()).toBeTruthy()
    await restart()
    expect((await list()).find((e) => e.manifest.id === 'e2e-ext')?.enabled).toBe(false)

    // An enabled one stays enabled.
    expect((await authed.post('/api/extensions/e2e-ext/enabled', { data: { enabled: true } })).ok()).toBeTruthy()
    await restart()
    expect((await list()).find((e) => e.manifest.id === 'e2e-ext')?.enabled).toBe(true)

    // Uninstall leaves nothing behind.
    expect((await authed.post('/api/extensions/e2e-ext/uninstall')).ok()).toBeTruthy()
    expect(await list()).toEqual([])
  } finally {
    await authed.dispose()
    await anonymous.dispose()
    await server.stop()
    await zipServer.stop()
    removeTempRoot(root)
    removeTempRoot(zipDir)
  }
})

