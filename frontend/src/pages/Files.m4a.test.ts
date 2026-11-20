import { describe, it, expect, beforeEach, vi } from 'vitest'

const api = vi.hoisted(() => ({
  isWailsRuntime: vi.fn(() => false),
  ListDownloadedFiles: vi.fn(),
  DeleteFile: vi.fn(),
  OpenDownloadFolder: vi.fn(),
  IsConverterAvailable: vi.fn(),
  FetchAndEmbedLyricsMultiple: vi.fn(),
  OpenFLACFilesDialog: vi.fn(),
  SelectFolderForConversion: vi.fn(),
  AnalyzeMultiple: vi.fn(),
  AnalyzeBatch: vi.fn(),
  CancelAnalyzeBatch: vi.fn(),
  ExportAnalysisReport: vi.fn(),
}))
vi.mock('../lib/api', () => api)

const drop = vi.hoisted(() => ({ cb: null as null | ((x: number, y: number, paths: string[]) => void) }))
vi.mock('../lib/runtime', () => ({
  onNativeFileDrop: (cb: (x: number, y: number, paths: string[]) => void) => {
    drop.cb = cb
    return () => {}
  },
}))

const FILES = [
  { path: '/m/a.flac', name: 'a.flac', size: 10, modTime: '2024-01-01T00:00:00Z', title: 'A', artist: 'X', album: 'Y', format: 'flac' },
  { path: '/m/b.m4a', name: 'b.m4a', size: 20, modTime: '2024-01-02T00:00:00Z', title: 'B', artist: 'X', album: 'Y', format: 'm4a', codec: 'eac3' },
  { path: '/m/c.m4a', name: 'c.m4a', size: 30, modTime: '2024-01-03T00:00:00Z', title: 'C', artist: 'X', album: 'Y', format: 'm4a', codec: 'aac' },
]

vi.setConfig({ testTimeout: 30000 })

async function mountFiles() {
  const { mount, flushSync, tick } = await import('svelte')
  const { default: Files } = await import('./Files.svelte')
  const target = document.createElement('div')
  document.body.appendChild(target)
  mount(Files, { target })
  const settle = async () => { for (let i = 0; i < 6; i++) await tick(); flushSync() }
  await settle()
  return { target, settle }
}

const rowOf = (target: HTMLElement, name: string) =>
  [...target.querySelectorAll('.table-row')].find((r) => r.querySelector('.file-path')?.textContent === name) as HTMLElement

describe('Files page with .m4a entries', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
    drop.cb = null
    api.ListDownloadedFiles.mockResolvedValue(FILES)
    api.IsConverterAvailable.mockResolvedValue(false)
    api.DeleteFile.mockResolvedValue(undefined)
    api.AnalyzeMultiple.mockResolvedValue([])
    api.AnalyzeBatch.mockResolvedValue({ results: [], cancelled: false })
  })

  it('badges Atmos and plain m4a, and leaves FLAC unbadged', async () => {
    const { target } = await mountFiles()
    expect(rowOf(target, 'a.flac').querySelector('.format-badge')).toBeNull()
    expect(rowOf(target, 'b.m4a').querySelector('.format-badge.atmos')?.textContent?.trim()).toBe('Dolby Atmos')
    const plain = rowOf(target, 'c.m4a').querySelector('.format-badge.m4a')
    expect(plain?.textContent?.trim()).toBe('M4A')
  })

  it('deletes an .m4a through DeleteFile after confirmation', async () => {
    const { target, settle } = await mountFiles()
    ;(rowOf(target, 'b.m4a').querySelector('.file-btn.danger') as HTMLButtonElement).click()
    await settle()
    ;(document.querySelector('.btn-confirm') as HTMLButtonElement).click()
    await settle()
    expect(api.DeleteFile).toHaveBeenCalledWith('/m/b.m4a')
  })

  it('accepts dropped .m4a files for analysis and still ignores other types', async () => {
    const { settle } = await mountFiles()
    drop.cb?.(0, 0, ['/x/one.m4a', '/x/two.mp3', '/x/three.FLAC'])
    await settle()
    const call = [...api.AnalyzeMultiple.mock.calls, ...api.AnalyzeBatch.mock.calls].at(-1)
    const paths = (call?.length === 2 ? call[1] : call?.[0]) as string[]
    expect(paths).toEqual(['/x/one.m4a', '/x/three.FLAC'])
  })
})
