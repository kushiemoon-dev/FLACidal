import { describe, it, expect, beforeEach, vi } from 'vitest'

const batch = vi.hoisted(() => ({
  start: vi.fn(),
  onProgress: null as null | ((d: number, t: number, r: unknown) => void),
  resolve: null as null | ((o: unknown) => void),
  cancel: vi.fn(async () => true),
}))
vi.mock('../lib/analyzeBatch', () => ({
  MAX_ANALYZE_BATCH: 5000,
  startAnalyzeBatch: (paths: string[], onProgress: (d: number, t: number, r: unknown) => void) => {
    batch.start(paths)
    batch.onProgress = onProgress
    return { done: new Promise((res) => { batch.resolve = res }), cancel: batch.cancel }
  },
}))

const row = (p: string) => ({ filePath: p, fileName: p, verdict: 'lossless', verdictLabel: 'True Lossless', isTrueLossless: true, confidence: 90, spectrumCutoff: 22000, expectedCutoff: 22050, details: '', sampleRate: 44100, bitDepth: 16 })

vi.setConfig({ testTimeout: 30000 })

async function mountModal(files: string[]) {
  const { mount, flushSync, tick } = await import('svelte')
  const { default: Modal } = await import('./AnalysisModal.svelte')
  const target = document.createElement('div')
  document.body.appendChild(target)
  mount(Modal, { target, props: { files, onClose: () => {} } })
  const settle = async () => { for (let i = 0; i < 6; i++) await tick(); flushSync() }
  await settle()
  return { target, settle }
}

describe('AnalysisModal batch flow', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
  })

  it('shows progress while analyzing and can be cancelled', async () => {
    const { target, settle } = await mountModal(['/a.flac', '/b.flac'])
    expect(batch.start).toHaveBeenCalledWith(['/a.flac', '/b.flac'])

    batch.onProgress?.(1, 2, row('/a.flac'))
    await settle()
    expect((target.querySelector('progress') as HTMLProgressElement).getAttribute('aria-valuenow')).toBe('1')
    expect(target.querySelector('.progress-text')?.textContent).toContain('1')

    ;(target.querySelector('.btn-cancel-analysis') as HTMLButtonElement).click()
    await settle()
    expect(batch.cancel).toHaveBeenCalledTimes(1)

    batch.resolve?.({ results: [row('/a.flac')], cancelled: true })
    await settle()
    expect(target.querySelector('progress')).toBeNull()
    expect(target.querySelector('.summary-section')).not.toBeNull()
  })

  it('shows the ceiling message and skips the server above 5000 files', async () => {
    const { target } = await mountModal(Array.from({ length: 5001 }, (_, i) => `/m/${i}.flac`))
    expect(batch.start).not.toHaveBeenCalled()
    expect(target.querySelector('.error-state')?.textContent).toContain('5001')
  })
})
