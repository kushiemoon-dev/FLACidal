import fs from 'node:fs'
import { test, expect } from '@playwright/test'
import { seedFlac, setFolder } from './helpers'
import { TOKEN, TOKEN_URL } from './servers'

interface BatchBody {
  results: Array<{ filePath: string; verdict: string; details: string }>
  cancelled: boolean
  total: number
}

test.describe('Batch analysis on the real server', () => {
  let dir: string

  test.beforeEach(async ({ request }) => {
    dir = seedFlac(['alpha.flac'])
    await setFolder(request, dir)
  })

  test.afterEach(() => {
    fs.rmSync(dir, { recursive: true, force: true })
  })

  test('a mixed batch keeps one row per path and shows the confinement error', async ({ request }) => {
    const inside = `${dir}/alpha.flac`
    const res = await request.post('/api/analyze/batch', {
      data: { batchId: 'e2e-mixed', paths: [inside, '/etc/hostname'] },
    })
    expect(res.status()).toBe(200)
    const body = (await res.json()) as BatchBody
    expect(body.total).toBe(2)
    expect(body.cancelled).toBe(false)
    expect(body.results).toHaveLength(2)
    const outside = body.results.find((r) => r.filePath === '/etc/hostname')
    expect(outside?.verdict).toBe('error')
    expect(outside?.details).not.toBe('')
    expect(body.results.some((r) => r.filePath === inside)).toBe(true)
  })

  test('refuses a batch above the ceiling with 413 before doing any work', async ({ request }) => {
    const paths = Array.from({ length: 5001 }, (_, i) => `${dir}/${i}.flac`)
    const res = await request.post('/api/analyze/batch', { data: { batchId: 'e2e-big', paths } })
    expect(res.status()).toBe(413)
    expect((await res.json()).error).toContain('maximum 5000')
    const multiple = await request.post('/api/analyze/multiple', { data: { paths } })
    expect(multiple.status()).toBe(413)
  })

  test('cancelling an unknown batch is a 404', async ({ request }) => {
    const res = await request.post('/api/analyze/batch/nope/cancel')
    expect(res.status()).toBe(404)
  })

  test('the report comes back as a CSV download with the formula neutralised', async ({ request }) => {
    const res = await request.post('/api/analyze/report?format=csv', {
      data: { results: [{ filePath: '/m/=cmd.flac', fileName: '=cmd.flac', verdict: 'lossless', verdictLabel: 'True Lossless' }] },
    })
    expect(res.status()).toBe(200)
    expect(res.headers()['content-disposition']).toContain('analysis_report.csv')
    const text = await res.text()
    expect(text.split('\n')[0]).toContain('path,file,verdict')
    expect(text).toContain("'=cmd.flac")
    const bad = await request.post('/api/analyze/report?format=xml', { data: { results: [{ filePath: '/a' }] } })
    expect(bad.status()).toBe(400)
  })
})

test.describe('Batch analysis routes need the token', () => {
  test.use({ baseURL: TOKEN_URL })

  for (const [method, route] of [
    ['POST', '/api/analyze/batch'],
    ['POST', '/api/analyze/batch/x/cancel'],
    ['POST', '/api/analyze/report?format=csv'],
  ] as const) {
    test(`${method} ${route} answers 401 without a token`, async ({ request }) => {
      const res = await request.fetch(route, { method, data: {} })
      expect(res.status()).toBe(401)
      const withToken = await request.fetch(route, { method, data: {}, headers: { Authorization: `Bearer ${TOKEN}` } })
      expect(withToken.status()).not.toBe(401)
    })
  }
})
