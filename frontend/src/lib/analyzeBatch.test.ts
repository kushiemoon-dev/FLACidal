import { describe, it, expect, beforeEach, vi } from 'vitest'

const api = vi.hoisted(() => ({ AnalyzeBatch: vi.fn(), CancelAnalyzeBatch: vi.fn() }))
vi.mock('./api', () => api)

const ws = vi.hoisted(() => ({
  handlers: new Map<string, (p: unknown) => void>(),
  unsub: vi.fn(),
}))
vi.mock('./websocket', () => ({
  EventsOn: (name: string, cb: (p: unknown) => void) => {
    ws.handlers.set(name, cb)
    return ws.unsub
  },
}))

import { startAnalyzeBatch, MAX_ANALYZE_BATCH } from './analyzeBatch'

const row = (p: string) => ({ filePath: p, fileName: p, verdict: 'lossless' })

describe('startAnalyzeBatch', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ws.handlers.clear()
  })

  it('subscribes before the call, forwards only its own batch, and unsubscribes at the end', async () => {
    let release!: (v: unknown) => void
    api.AnalyzeBatch.mockImplementation((id: string) => {
      expect(ws.handlers.has('analyze-progress')).toBe(true)
      void id
      return new Promise((r) => { release = r })
    })
    const seen: Array<[number, number, unknown]> = []
    const run = startAnalyzeBatch(['/a', '/b'], (done, total, result) => seen.push([done, total, result]))
    const batchId = api.AnalyzeBatch.mock.calls[0][0] as string

    const emit = ws.handlers.get('analyze-progress')!
    emit({ batchId, done: 1, total: 2, result: row('/a') })
    emit({ batchId: 'other', done: 1, total: 9, result: row('/x') })
    expect(seen).toEqual([[1, 2, row('/a')]])

    release({ results: [row('/a')], cancelled: true })
    const outcome = await run.done
    expect(outcome).toEqual({ results: [row('/a')], cancelled: true })
    expect(ws.unsub).toHaveBeenCalledTimes(1)
  })

  it('unsubscribes when the call fails and rethrows', async () => {
    api.AnalyzeBatch.mockRejectedValue(new Error('too many files: 5001 (maximum 5000)'))
    const run = startAnalyzeBatch(['/a'], () => {})
    await expect(run.done).rejects.toThrow('too many files')
    expect(ws.unsub).toHaveBeenCalledTimes(1)
  })

  it('cancel calls CancelAnalyzeBatch with the same id', async () => {
    api.AnalyzeBatch.mockReturnValue(new Promise(() => {}))
    api.CancelAnalyzeBatch.mockResolvedValue(true)
    const run = startAnalyzeBatch(['/a'], () => {})
    await run.cancel()
    expect(api.CancelAnalyzeBatch).toHaveBeenCalledWith(api.AnalyzeBatch.mock.calls[0][0])
  })

  it('exposes the shared ceiling', () => {
    expect(MAX_ANALYZE_BATCH).toBe(5000)
  })

  it('still works where crypto.randomUUID is missing (plain http on a LAN address)', async () => {
    vi.stubGlobal('crypto', { getRandomValues: (a: Uint8Array) => a.fill(7) })
    try {
      api.AnalyzeBatch.mockReturnValue(new Promise(() => {}))
      expect(() => startAnalyzeBatch(['/a'], () => {})).not.toThrow()
      const id = api.AnalyzeBatch.mock.calls[0][0] as string
      expect(id).toMatch(/^[0-9a-f-]{16,}$/)
    } finally {
      vi.unstubAllGlobals()
    }
  })
})
