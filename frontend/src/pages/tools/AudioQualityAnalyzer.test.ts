import { describe, it, expect, beforeEach, vi } from 'vitest'

const api = vi.hoisted(() => ({
  ExportAnalysisReport: vi.fn(),
  OpenFLACFilesDialog: vi.fn(),
  SelectFolderForAnalysis: vi.fn(),
  isWailsRuntime: vi.fn(() => false),
}))
vi.mock('../../lib/api', () => api)

const batch = vi.hoisted(() => ({
  start: vi.fn(),
  onProgress: null as null | ((d: number, t: number, r: unknown) => void),
  resolve: null as null | ((o: unknown) => void),
  reject: null as null | ((e: unknown) => void),
  cancel: vi.fn(async () => true),
}))
vi.mock('../../lib/analyzeBatch', () => ({
  MAX_ANALYZE_BATCH: 5000,
  startAnalyzeBatch: (paths: string[], onProgress: (d: number, t: number, r: unknown) => void) => {
    batch.start(paths)
    batch.onProgress = onProgress
    return {
      done: new Promise((res, rej) => { batch.resolve = res; batch.reject = rej }),
      cancel: batch.cancel,
    }
  },
}))

const drop = vi.hoisted(() => ({ cb: null as null | ((x: number, y: number, p: string[]) => void) }))
vi.mock('../../lib/runtime', () => ({
  onNativeFileDrop: (cb: (x: number, y: number, p: string[]) => void) => { drop.cb = cb; return () => {} },
}))

const toast = vi.hoisted(() => ({ showKey: vi.fn(), showMsg: vi.fn() }))
vi.mock('../../stores/toast', () => ({ toastStore: toast }))

const ok = (p: string) => ({ filePath: p, fileName: p.split('/').pop(), verdict: 'lossless', verdictLabel: 'True Lossless', isTrueLossless: true, confidence: 90, sampleRate: 44100, bitDepth: 16, details: '' })
const bad = (p: string) => ({ filePath: p, fileName: p.split('/').pop(), verdict: 'error', verdictLabel: 'Error', isTrueLossless: false, confidence: 0, sampleRate: 0, bitDepth: 0, details: 'cannot decode' })

vi.setConfig({ testTimeout: 30000 })

async function mountPage() {
  const { mount, flushSync, tick } = await import('svelte')
  const { default: Page } = await import('./AudioQualityAnalyzer.svelte')
  const target = document.createElement('div')
  document.body.appendChild(target)
  mount(Page, { target })
  const settle = async () => { for (let i = 0; i < 6; i++) await tick(); flushSync() }
  await settle()
  return { target, settle }
}

describe('AudioQualityAnalyzer batch flow', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
    drop.cb = null
    api.ExportAnalysisReport.mockResolvedValue('')
  })

  it('shows progress, lists errors apart, and keeps the final results', async () => {
    const { target, settle } = await mountPage()
    drop.cb?.(0, 0, ['/m/a.flac', '/m/bad.flac', '/m/c.flac'])
    await settle()
    expect(batch.start).toHaveBeenCalledWith(['/m/a.flac', '/m/bad.flac', '/m/c.flac'])
    expect((target.querySelector('.btn-export-csv') as HTMLButtonElement).disabled).toBe(true)

    batch.onProgress?.(1, 3, ok('/m/a.flac'))
    batch.onProgress?.(2, 3, bad('/m/bad.flac'))
    await settle()
    const bar = target.querySelector('progress.analyze-progress') as HTMLProgressElement
    expect(bar.getAttribute('aria-valuenow')).toBe('2')
    expect(bar.getAttribute('aria-valuemax')).toBe('3')
    expect(target.querySelector('.progress-text')?.textContent).toContain('2')
    expect(target.querySelectorAll('.table-row')).toHaveLength(1)
    expect(target.querySelector('.errors-panel')?.textContent).toContain('bad.flac')
    expect(target.querySelector('.errors-panel')?.textContent).toContain('cannot decode')

    batch.resolve?.({ results: [ok('/m/a.flac'), bad('/m/bad.flac'), ok('/m/c.flac')], cancelled: false })
    await settle()
    expect(target.querySelector('progress.analyze-progress')).toBeNull()
    expect(target.querySelectorAll('.table-row')).toHaveLength(2)
    expect(target.querySelectorAll('.error-row')).toHaveLength(1)
  })

  it('cancel keeps the results already received and says how many', async () => {
    const { target, settle } = await mountPage()
    drop.cb?.(0, 0, ['/m/a.flac', '/m/b.flac', '/m/c.flac'])
    await settle()
    batch.onProgress?.(1, 3, ok('/m/a.flac'))
    await settle()

    ;(target.querySelector('.btn-cancel-analysis') as HTMLButtonElement).click()
    await settle()
    expect(batch.cancel).toHaveBeenCalledTimes(1)

    batch.resolve?.({ results: [ok('/m/a.flac')], cancelled: true })
    await settle()
    expect(target.querySelectorAll('.table-row')).toHaveLength(1)
    const note = target.querySelector('.cancelled-note')?.textContent ?? ''
    expect(note).toContain('1')
    expect(note).toContain('3')
  })

  it('refuses more than 5000 files without calling the server', async () => {
    await mountPage()
    drop.cb?.(0, 0, Array.from({ length: 5001 }, (_, i) => `/m/${i}.flac`))
    expect(batch.start).not.toHaveBeenCalled()
    expect(toast.showKey).toHaveBeenCalledWith('tools.analyzer.tooMany', { count: 5001, max: 5000 }, 'error')
  })

  it('reports a failed call with a visible toast', async () => {
    const { settle } = await mountPage()
    drop.cb?.(0, 0, ['/m/a.flac'])
    await settle()
    batch.reject?.(new Error('network down'))
    await settle()
    expect(toast.showKey).toHaveBeenCalledWith('tools.analyzer.failed', { error: 'network down' }, 'error')
  })

  it('exports every result, errors included, as CSV or JSON', async () => {
    const { target, settle } = await mountPage()
    drop.cb?.(0, 0, ['/m/a.flac', '/m/bad.flac'])
    await settle()
    const all = [ok('/m/a.flac'), bad('/m/bad.flac')]
    batch.resolve?.({ results: all, cancelled: false })
    await settle()

    ;(target.querySelector('.btn-export-csv') as HTMLButtonElement).click()
    await settle()
    expect(api.ExportAnalysisReport).toHaveBeenLastCalledWith(all, 'csv')
    ;(target.querySelector('.btn-export-json') as HTMLButtonElement).click()
    await settle()
    expect(api.ExportAnalysisReport).toHaveBeenLastCalledWith(all, 'json')
  })

  it('re-enables Cancel when the server did not know the batch yet', async () => {
    const { target, settle } = await mountPage()
    drop.cb?.(0, 0, ['/m/a.flac', '/m/b.flac'])
    await settle()
    batch.cancel.mockResolvedValueOnce(false)

    const btn = target.querySelector('.btn-cancel-analysis') as HTMLButtonElement
    btn.click()
    await settle()
    expect(btn.disabled).toBe(false)

    batch.cancel.mockRejectedValueOnce(new Error('404'))
    btn.click()
    await settle()
    expect(btn.disabled).toBe(false)
  })
})
