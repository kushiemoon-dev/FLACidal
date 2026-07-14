import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

// wailsjs/go/app/App.js is a generated file that calls window.go.app.App.*
// under the hood; mock it directly so Wails-mode tests never touch a real
// window.go.
const wailsMock = {
  GetAppVersion: vi.fn(),
  GetDownloadFolder: vi.fn(),
  IsQueuePaused: vi.fn(),
  CancelDownload: vi.fn(),
  QueueDownloads: vi.fn(),
  AnalyzeMultiple: vi.fn(),
  OpenFLACFilesDialog: vi.fn(),
}
vi.mock('../../wailsjs/go/app/App.js', () => wailsMock)

function setWailsRuntime() {
  ;(window as any).go = { app: { App: {} } }
  ;(window as any).runtime = {}
}

function clearWailsRuntime() {
  delete (window as any).go
  delete (window as any).runtime
}

describe('isWailsRuntime', () => {
  afterEach(() => {
    vi.resetModules()
    clearWailsRuntime()
  })

  it('returns true when window.go.app.App and window.runtime are present', async () => {
    setWailsRuntime()
    const { isWailsRuntime } = await import('./api')
    expect(isWailsRuntime()).toBe(true)
  })

  it('returns false in a plain browser (no window.go)', async () => {
    const { isWailsRuntime } = await import('./api')
    expect(isWailsRuntime()).toBe(false)
  })

  it('caches the result after the first call', async () => {
    const { isWailsRuntime } = await import('./api')
    expect(isWailsRuntime()).toBe(false)

    // Flipping window.go after the first call must not change the cached result.
    setWailsRuntime()
    expect(isWailsRuntime()).toBe(false)
  })
})

describe('API routing — Wails mode', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
    setWailsRuntime()
  })
  afterEach(() => {
    clearWailsRuntime()
  })

  it('GetAppVersion calls the Wails binding and never touches fetch', async () => {
    wailsMock.GetAppVersion.mockResolvedValue('4.12.0')
    const fetchSpy = vi.spyOn(globalThis, 'fetch')

    const { GetAppVersion } = await import('./api')
    const version = await GetAppVersion()

    expect(version).toBe('4.12.0')
    expect(wailsMock.GetAppVersion).toHaveBeenCalledOnce()
    expect(fetchSpy).not.toHaveBeenCalled()
  })

  it('QueueDownloads forwards all 5 args positionally to the Wails binding', async () => {
    wailsMock.QueueDownloads.mockResolvedValue(3)
    const tracks = [{ id: 1 }]

    const { QueueDownloads } = await import('./api')
    const queued = await QueueDownloads(tracks, '/music', 'Discovery', 'content-1', 'album')

    expect(queued).toBe(3)
    expect(wailsMock.QueueDownloads).toHaveBeenCalledWith(tracks, '/music', 'Discovery', 'content-1', 'album')
  })

  it('OpenFLACFilesDialog delegates to the native Wails dialog', async () => {
    wailsMock.OpenFLACFilesDialog.mockResolvedValue(['/music/a.flac'])

    const { OpenFLACFilesDialog } = await import('./api')
    const paths = await OpenFLACFilesDialog()

    expect(paths).toEqual(['/music/a.flac'])
    expect(wailsMock.OpenFLACFilesDialog).toHaveBeenCalledOnce()
  })
})

describe('API routing — browser mode', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
    clearWailsRuntime()
  })

  function mockFetchOnce(body: unknown, ok = true) {
    const fetchMock = vi.fn().mockResolvedValue({
      ok,
      status: ok ? 200 : 500,
      statusText: ok ? 'OK' : 'Internal Server Error',
      json: async () => body,
      blob: async () => new Blob([JSON.stringify(body)]),
    })
    vi.stubGlobal('fetch', fetchMock)
    return fetchMock
  }

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('GetAppVersion fetches GET /api/version and unwraps .version', async () => {
    const fetchMock = mockFetchOnce({ version: '1.0.0' })

    const { GetAppVersion } = await import('./api')
    const version = await GetAppVersion()

    expect(version).toBe('1.0.0')
    expect(fetchMock).toHaveBeenCalledWith('/api/version', undefined)
  })

  it('IsQueuePaused fetches GET /api/downloads/paused and unwraps .paused', async () => {
    mockFetchOnce({ paused: true })

    const { IsQueuePaused } = await import('./api')
    expect(await IsQueuePaused()).toBe(true)
  })

  it('QueueDownloads POSTs {tracks,outputDir,contentName} and unwraps .queued', async () => {
    const fetchMock = mockFetchOnce({ queued: 3 })
    const tracks = [{ id: 1 }]

    const { QueueDownloads } = await import('./api')
    const queued = await QueueDownloads(tracks, '/music', 'Discovery', 'content-1', 'album')

    expect(queued).toBe(3)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/downloads/queue')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body)).toEqual({ tracks, outputDir: '/music', contentName: 'Discovery' })
  })

  it('AnalyzeMultiple normalizes the REST shape to the AnalysisResult shape', async () => {
    mockFetchOnce([
      { fileName: 'a.flac', isUpscaled: false, confidence: 90, spectralCutoff: 22000, verdict: 'pass', verdictLabel: 'Lossless', message: 'Authentic lossless', sampleRate: 44100, bitDepth: 16 },
    ])

    const { AnalyzeMultiple } = await import('./api')
    const [result] = await AnalyzeMultiple(['/music/a.flac'])

    expect(result).toEqual({
      filePath: '/music/a.flac',
      fileName: 'a.flac',
      isTrueLossless: true,
      confidence: 90,
      spectrumCutoff: 22000,
      expectedCutoff: 0,
      verdict: 'pass',
      verdictLabel: 'Lossless',
      details: 'Authentic lossless',
      sampleRate: 44100,
      bitDepth: 16,
    })
  })

  it('apiFetch throws with the server-provided error message on a non-ok response', async () => {
    mockFetchOnce({ error: 'no output directory specified' }, false)

    const { QueueSingleDownload } = await import('./api')
    await expect(QueueSingleDownload(1, '', 'Track', 'Artist')).rejects.toThrow('no output directory specified')
  })

  it('OpenFLACFilesDialog resolves to [] instead of throwing (native dialog has no browser equivalent)', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch')
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    const { OpenFLACFilesDialog } = await import('./api')
    const paths = await OpenFLACFilesDialog()

    expect(paths).toEqual([])
    expect(fetchSpy).not.toHaveBeenCalled()
    warnSpy.mockRestore()
  })

  it('SelectDownloadFolder resolves to "" instead of throwing', async () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    const { SelectDownloadFolder } = await import('./api')
    expect(await SelectDownloadFolder()).toBe('')
    warnSpy.mockRestore()
  })
})
