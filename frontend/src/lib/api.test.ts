import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

// wailsjs/go/app/App.js is generated code that calls into window.go.app.App.*
// under the hood, so it's mocked directly here, that way the Wails-mode
// tests never touch a real window.go.
const wailsMock = {
  GetAppVersion: vi.fn(),
  GetDownloadFolder: vi.fn(),
  IsQueuePaused: vi.fn(),
  CancelDownload: vi.fn(),
  QueueDownloads: vi.fn(),
  AnalyzeMultiple: vi.fn(),
  OpenFLACFilesDialog: vi.fn(),
  GetUpdateStatus: vi.fn(),
  DownloadAndInstallUpdate: vi.fn(),
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

  it('reports true once both window.go.app.App and window.runtime exist', async () => {
    setWailsRuntime()
    const { isWailsRuntime } = await import('./api')
    expect(isWailsRuntime()).toBe(true)
  })

  it('reports false in a plain browser where window.go is absent', async () => {
    const { isWailsRuntime } = await import('./api')
    expect(isWailsRuntime()).toBe(false)
  })

  it('remembers its first result on subsequent calls', async () => {
    const { isWailsRuntime } = await import('./api')
    expect(isWailsRuntime()).toBe(false)

    setWailsRuntime()
    expect(isWailsRuntime()).toBe(false)
  })
})

describe('API call routing in Wails mode', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
    setWailsRuntime()
  })
  afterEach(() => {
    clearWailsRuntime()
  })

  it('GetAppVersion goes through the Wails binding and skips fetch entirely', async () => {
    wailsMock.GetAppVersion.mockResolvedValue('4.12.0')
    const fetchSpy = vi.spyOn(globalThis, 'fetch')

    const { GetAppVersion } = await import('./api')
    const version = await GetAppVersion()

    expect(version).toBe('4.12.0')
    expect(wailsMock.GetAppVersion).toHaveBeenCalledOnce()
    expect(fetchSpy).not.toHaveBeenCalled()
  })

  it('QueueDownloads passes all five positional arguments through to the Wails binding', async () => {
    wailsMock.QueueDownloads.mockResolvedValue(3)
    const tracks = [{ id: 1 }]

    const { QueueDownloads } = await import('./api')
    const queued = await QueueDownloads(tracks, '/music', 'Discovery', 'content-1', 'album')

    expect(queued).toBe(3)
    expect(wailsMock.QueueDownloads).toHaveBeenCalledWith(tracks, '/music', 'Discovery', 'content-1', 'album')
  })

  it('OpenFLACFilesDialog hands off to the native Wails dialog', async () => {
    wailsMock.OpenFLACFilesDialog.mockResolvedValue(['/music/a.flac'])

    const { OpenFLACFilesDialog } = await import('./api')
    const paths = await OpenFLACFilesDialog()

    expect(paths).toEqual(['/music/a.flac'])
    expect(wailsMock.OpenFLACFilesDialog).toHaveBeenCalledOnce()
  })

  it('GetUpdateStatus goes through the Wails binding', async () => {
    const status = { hasUpdate: true, currentVersion: '4.9.0', latestVersion: '4.10.0', versionsBehind: 1, blocked: false, releaseUrl: 'https://example.com' }
    wailsMock.GetUpdateStatus.mockResolvedValue(status)

    const { GetUpdateStatus } = await import('./api')
    expect(await GetUpdateStatus()).toEqual(status)
    expect(wailsMock.GetUpdateStatus).toHaveBeenCalledOnce()
  })

  it('DownloadAndInstallUpdate hands off to the Wails binding', async () => {
    wailsMock.DownloadAndInstallUpdate.mockResolvedValue(undefined)

    const { DownloadAndInstallUpdate } = await import('./api')
    await DownloadAndInstallUpdate()

    expect(wailsMock.DownloadAndInstallUpdate).toHaveBeenCalledOnce()
  })
})

describe('API call routing in browser mode', () => {
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

  it('GetAppVersion issues a GET to /api/version and unwraps the .version field', async () => {
    const fetchMock = mockFetchOnce({ version: '1.0.0' })

    const { GetAppVersion } = await import('./api')
    const version = await GetAppVersion()

    expect(version).toBe('1.0.0')
    expect(fetchMock).toHaveBeenCalledWith('/api/version', undefined)
  })

  it('IsQueuePaused issues a GET to /api/downloads/paused and unwraps the .paused field', async () => {
    mockFetchOnce({ paused: true })

    const { IsQueuePaused } = await import('./api')
    expect(await IsQueuePaused()).toBe(true)
  })

  it('QueueDownloads POSTs {tracks,outputDir,contentName} and unwraps the .queued field', async () => {
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

  it('AnalyzeMultiple converts the REST response shape into the AnalysisResult shape', async () => {
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

  it('apiFetch throws using the server-supplied error message when the response is not ok', async () => {
    mockFetchOnce({ error: 'no output directory specified' }, false)

    const { QueueSingleDownload } = await import('./api')
    await expect(QueueSingleDownload(1, '', 'Track', 'Artist')).rejects.toThrow('no output directory specified')
  })

  it('OpenFLACFilesDialog resolves to [] rather than throwing, since the native dialog has no browser counterpart', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch')
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    const { OpenFLACFilesDialog } = await import('./api')
    const paths = await OpenFLACFilesDialog()

    expect(paths).toEqual([])
    expect(fetchSpy).not.toHaveBeenCalled()
    warnSpy.mockRestore()
  })

  it('SelectDownloadFolder resolves to "" rather than throwing', async () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    const { SelectDownloadFolder } = await import('./api')
    expect(await SelectDownloadFolder()).toBe('')
    warnSpy.mockRestore()
  })

  it('GetUpdateStatus warns and resolves to undefined, forced-update blocking does not apply headless', async () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    const { GetUpdateStatus } = await import('./api')
    expect(await GetUpdateStatus()).toBeUndefined()
    expect(warnSpy).toHaveBeenCalled()
    warnSpy.mockRestore()
  })

  it('DownloadAndInstallUpdate warns rather than throwing', async () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    const { DownloadAndInstallUpdate } = await import('./api')
    await DownloadAndInstallUpdate()
    expect(warnSpy).toHaveBeenCalled()
    warnSpy.mockRestore()
  })
})

describe('compareVersionParts', () => {
  it('treats a higher minor version as greater regardless of digit count', async () => {
    const { compareVersionParts } = await import('./api')
    expect(compareVersionParts('4.10.0', '4.9.0')).toBeGreaterThan(0)
  })

  it('treats a lower version as lesser', async () => {
    const { compareVersionParts } = await import('./api')
    expect(compareVersionParts('4.9.0', '4.10.0')).toBeLessThan(0)
  })

  it('treats equal versions as equal', async () => {
    const { compareVersionParts } = await import('./api')
    expect(compareVersionParts('4.10.0', '4.10.0')).toBe(0)
  })

  it('treats a missing trailing part as zero', async () => {
    const { compareVersionParts } = await import('./api')
    expect(compareVersionParts('4.10', '4.10.0')).toBe(0)
  })
})

describe('CheckForUpdate in browser mode', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
    clearWailsRuntime()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  function mockGithubRelease(body: unknown) {
    const fetchMock = vi.fn().mockImplementation((url: string) => {
      if (url === 'https://api.github.com/repos/kushiemoon-dev/flacidal/releases/latest') {
        return Promise.resolve({ ok: true, status: 200, json: async () => body })
      }
      if (url === '/api/version') {
        return Promise.resolve({ ok: true, status: 200, json: async () => ({ version: '4.9.0' }) })
      }
      throw new Error(`unexpected fetch: ${url}`)
    })
    vi.stubGlobal('fetch', fetchMock)
    return fetchMock
  }

  it('flags an update using semver comparison, not lexicographic', async () => {
    mockGithubRelease({ tag_name: 'v4.10.0', html_url: 'https://example.com/release', assets: [] })

    const { CheckForUpdate } = await import('./api')
    const info = await CheckForUpdate()

    expect(info.hasUpdate).toBe(true)
    expect(info.version).toBe('4.10.0')
  })

  it('selects the asset matching the browser platform instead of assets[0]', async () => {
    vi.stubGlobal('navigator', { userAgent: 'Macintosh; Intel Mac OS X 10_15' })
    mockGithubRelease({
      tag_name: 'v4.10.0',
      html_url: 'https://example.com/release',
      assets: [
        { name: 'flacidal.exe', browser_download_url: 'https://example.com/flacidal.exe' },
        { name: 'flacidal.dmg', browser_download_url: 'https://example.com/flacidal.dmg' },
      ],
    })

    const { CheckForUpdate } = await import('./api')
    const info = await CheckForUpdate()

    expect(info.url).toBe('https://example.com/flacidal.dmg')
  })

  it('falls back to the release page URL on an Android UA instead of matching Linux', async () => {
    vi.stubGlobal('navigator', {
      userAgent: 'Mozilla/5.0 (Linux; Android 13; Pixel 7)',
      maxTouchPoints: 5,
    })
    mockGithubRelease({
      tag_name: 'v4.10.0',
      html_url: 'https://example.com/release',
      assets: [{ name: 'flacidal.AppImage', browser_download_url: 'https://example.com/flacidal.AppImage' }],
    })

    const { CheckForUpdate } = await import('./api')
    const info = await CheckForUpdate()

    expect(info.url).toBe('https://example.com/release')
  })

  it('falls back to the release page URL on an iPadOS UA instead of matching macOS', async () => {
    vi.stubGlobal('navigator', {
      userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_6) AppleWebKit/605.1.15',
      maxTouchPoints: 5,
    })
    mockGithubRelease({
      tag_name: 'v4.10.0',
      html_url: 'https://example.com/release',
      assets: [{ name: 'flacidal.dmg', browser_download_url: 'https://example.com/flacidal.dmg' }],
    })

    const { CheckForUpdate } = await import('./api')
    const info = await CheckForUpdate()

    expect(info.url).toBe('https://example.com/release')
  })
})
