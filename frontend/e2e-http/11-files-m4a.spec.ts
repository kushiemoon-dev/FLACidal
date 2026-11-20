import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { test, expect } from '@playwright/test'
import { setFolder } from './helpers'

test.describe('.m4a files on the real server', () => {
  let dir: string

  test.beforeEach(async ({ request }) => {
    dir = fs.mkdtempSync(path.join(os.tmpdir(), 'flacidal-e2e-m4a-'))
    // Arbitrary bytes: without a usable ffprobe result the entry is still listed, just without a codec.
    for (const n of ['song.m4a', 'song.mp3']) fs.writeFileSync(path.join(dir, n), 'not real audio')
    await setFolder(request, dir)
  })

  test.afterEach(() => {
    fs.rmSync(dir, { recursive: true, force: true })
  })

  test('lists the .m4a (and not the .mp3), then deletes it', async ({ request }) => {
    const list = await request.get('/api/files')
    const files = (await list.json()) as Array<{ name: string; format: string }>
    expect(files.map((f) => f.name)).toEqual(['song.m4a'])
    expect(files[0].format).toBe('m4a')

    const del = await request.delete(`/api/files?path=${encodeURIComponent(path.join(dir, 'song.m4a'))}`)
    expect(del.status()).toBe(200)
    expect(fs.existsSync(path.join(dir, 'song.m4a'))).toBe(false)
  })

  test('still refuses an .mp3 (400) and a path outside the folders (403)', async ({ request }) => {
    const mp3 = await request.delete(`/api/files?path=${encodeURIComponent(path.join(dir, 'song.mp3'))}`)
    expect(mp3.status()).toBe(400)
    expect(fs.existsSync(path.join(dir, 'song.mp3'))).toBe(true)

    const outside = path.join(os.tmpdir(), `outside-${Date.now()}.m4a`)
    fs.writeFileSync(outside, 'x')
    try {
      const res = await request.delete(`/api/files?path=${encodeURIComponent(outside)}`)
      expect(res.status()).toBe(403)
      expect(fs.existsSync(outside)).toBe(true)
    } finally {
      fs.rmSync(outside, { force: true })
    }
  })
})
