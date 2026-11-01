import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { test, expect } from '@playwright/test'
import { seedFlac, setFolder } from './helpers'
import { TOKEN_URL } from './servers'

// Everything here stays offline: the network flows are covered by the Go
// handler tests with mocked sources.
test.describe('Discography, artist assets and FFmpeg routes', () => {
  let dir: string
  let outside: string

  test.beforeEach(async ({ request }) => {
    dir = seedFlac(['alpha.flac'])
    outside = fs.mkdtempSync(path.join(os.tmpdir(), 'flacidal-e2e-outside-'))
    await setFolder(request, dir)
  })

  test.afterEach(() => {
    fs.rmSync(dir, { recursive: true, force: true })
    fs.rmSync(outside, { recursive: true, force: true })
  })

  test('an output folder outside the allowed ones is refused', async ({ request }) => {
    const queue = await request.post('/api/discography/queue', { data: { albumUrls: [], outputDir: outside } })
    expect(queue.status()).toBe(403)
    const assets = await request.post('/api/artists/assets', { data: { artistId: '1', artistName: 'A', outputDir: outside } })
    expect(assets.status()).toBe(403)
  })

  test('a URL that is not a Spotify discography is a readable 400', async ({ request }) => {
    const res = await request.post('/api/discography/expand', { data: { url: 'https://example.com/x' } })
    expect(res.status()).toBe(400)
    expect((await res.json()).error).toContain('Spotify')
  })

  test('invalid JSON is a 400', async ({ request }) => {
    const res = await request.post('/api/artists/assets', { data: '{nope', headers: { 'Content-Type': 'application/json' } })
    expect(res.status()).toBe(400)
  })

  test('all four routes need the token on the protected server', async ({ playwright }) => {
    const anon = await playwright.request.newContext({ baseURL: TOKEN_URL })
    for (const p of ['/api/discography/expand', '/api/discography/queue', '/api/artists/assets', '/api/ffmpeg/install']) {
      expect((await anon.post(p, { data: {} })).status(), p).toBe(401)
    }
    await anon.dispose()
  })
})
