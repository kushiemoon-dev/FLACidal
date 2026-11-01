import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import apiSource from './api.ts?raw'

type WailsWindow = { go?: unknown; runtime?: unknown }

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
  ;(window as unknown as WailsWindow).go = { app: { App: {} } }
  ;(window as unknown as WailsWindow).runtime = {}
}

function clearWailsRuntime() {
  delete (window as unknown as WailsWindow).go
  delete (window as unknown as WailsWindow).runtime
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
    expect(fetchMock).toHaveBeenCalledWith('/api/version', expect.objectContaining({ headers: expect.any(Headers) }))
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

  it('GetUpdateStatus resolves to null silently, forced-update blocking does not apply headless', async () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    const { GetUpdateStatus } = await import('./api')
    expect(await GetUpdateStatus()).toBeNull()
    expect(warnSpy).not.toHaveBeenCalled()
    warnSpy.mockRestore()
  })

  it.each(['OpenDownloadFolder', 'OpenConfigFolder', 'DownloadAndInstallUpdate'] as const)(
    '%s rejects with an explicit desktop-only error instead of a silent no-op',
    async (name) => {
      const api = await import('./api')
      const call = name === 'OpenDownloadFolder' ? api.OpenDownloadFolder('/x') : api[name]()
      await expect(call).rejects.toThrow('only available in the desktop app')
    }
  )
})

describe('server parity in browser mode', () => {
  const requestPick = vi.fn()
  vi.doMock('./filePicker', () => ({ requestPick }))

  function mockFetch(body: unknown) {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true, status: 200, statusText: 'OK',
      json: async () => body,
      blob: async () => new Blob([JSON.stringify(body)]),
    })
    vi.stubGlobal('fetch', fetchMock)
    return fetchMock
  }

  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
    clearWailsRuntime()
    vi.doMock('./filePicker', () => ({ requestPick }))
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('BrowseFolder and ListFlacInFolder call the browse routes', async () => {
    const fetchMock = mockFetch({ path: '/m', parent: '', entries: [], truncated: false })
    const { BrowseFolder, ListFlacInFolder } = await import('./api')
    await BrowseFolder('/m', 'tools', true)
    expect(fetchMock.mock.calls[0][0]).toBe('/api/browse?path=%2Fm&scope=tools&files=1')
    await ListFlacInFolder('/m')
    expect(fetchMock.mock.calls[1][0]).toBe('/api/browse/flac?path=%2Fm')
  })

  it('OpenFLACFilesDialog returns the picked files, [] when cancelled', async () => {
    const { OpenFLACFilesDialog } = await import('./api')
    requestPick.mockResolvedValueOnce(['/a.flac', '/b.flac'])
    expect(await OpenFLACFilesDialog()).toEqual(['/a.flac', '/b.flac'])
    expect(requestPick).toHaveBeenCalledWith({ kind: 'files', scope: 'tools' })
    requestPick.mockResolvedValueOnce(null)
    expect(await OpenFLACFilesDialog()).toEqual([])
  })

  it('SelectDownloadFolder browses anywhere by default, and honours the tools scope', async () => {
    const { SelectDownloadFolder } = await import('./api')
    requestPick.mockResolvedValueOnce(['/music'])
    expect(await SelectDownloadFolder()).toBe('/music')
    expect(requestPick).toHaveBeenLastCalledWith({ kind: 'folder', scope: 'anywhere' })
    requestPick.mockResolvedValueOnce(['/out'])
    await SelectDownloadFolder('tools')
    expect(requestPick).toHaveBeenLastCalledWith({ kind: 'folder', scope: 'tools' })
    requestPick.mockResolvedValueOnce(null)
    expect(await SelectDownloadFolder()).toBe('')
  })

  it.each(['SelectFolderForConversion', 'SelectFolderForAnalysis'] as const)(
    '%s picks a folder then lists its FLAC files, [] when cancelled',
    async (name) => {
      const fetchMock = mockFetch({ paths: ['/m/a.flac'], truncated: false })
      const api = await import('./api')
      requestPick.mockResolvedValueOnce(['/m'])
      expect(await api[name]()).toEqual(['/m/a.flac'])
      expect(requestPick).toHaveBeenCalledWith({ kind: 'folder', scope: 'tools' })
      expect(fetchMock.mock.calls[0][0]).toBe('/api/browse/flac?path=%2Fm')
      requestPick.mockResolvedValueOnce(null)
      expect(await api[name]()).toEqual([])
      expect(fetchMock).toHaveBeenCalledTimes(1)
    }
  )

  it('ExportConfig downloads /api/config/export through a temporary link', async () => {
    const fetchMock = mockFetch({ schemaVersion: 1 })
    const create = vi.fn(() => 'blob:x')
    const revoke = vi.fn()
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: create, revokeObjectURL: revoke }))
    let downloadName = ''
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      downloadName = this.download
    })
    const { ExportConfig } = await import('./api')
    await ExportConfig()
    expect(fetchMock.mock.calls[0][0]).toBe('/api/config/export')
    expect(downloadName).toBe('flacidal-config.json')
    expect(revoke).toHaveBeenCalledWith('blob:x')
  })

  it('ImportConfig sends the chosen file, then returns the reloaded config', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce({ ok: true, status: 200, statusText: 'OK', json: async () => ({ success: true }) })
      .mockResolvedValueOnce({ ok: true, status: 200, statusText: 'OK', json: async () => ({ language: 'de' }) })
    vi.stubGlobal('fetch', fetchMock)
    const file = new File(['{"schemaVersion":1}'], 'c.json', { type: 'application/json' })
    Object.defineProperty(file, 'text', { value: async () => '{"schemaVersion":1}' }) // jsdom lacks File.text
    vi.spyOn(HTMLInputElement.prototype, 'click').mockImplementation(function (this: HTMLInputElement) {
      Object.defineProperty(this, 'files', { value: [file] })
      this.dispatchEvent(new Event('change'))
    })
    const { ImportConfig } = await import('./api')
    const cfg = await ImportConfig()
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/config/import')
    expect(init.method).toBe('POST')
    expect(init.body).toBe('{"schemaVersion":1}')
    expect(fetchMock.mock.calls[1][0]).toBe('/api/config')
    expect(cfg).toEqual({ language: 'de' })
    expect(document.querySelector('input[type=file]')).toBeNull()
  })

  it('ImportConfig resolves undefined when the file chooser is cancelled', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    vi.spyOn(HTMLInputElement.prototype, 'click').mockImplementation(function (this: HTMLInputElement) {
      this.dispatchEvent(new Event('cancel'))
    })
    const { ImportConfig } = await import('./api')
    expect(await ImportConfig()).toBeUndefined()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('discography, artist assets and FFmpeg install hit their routes', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce({ ok: true, status: 200, statusText: 'OK', json: async () => ({ albums: ['u1'] }) })
      .mockResolvedValueOnce({ ok: true, status: 200, statusText: 'OK', json: async () => ({ queued: 4 }) })
      .mockResolvedValueOnce({ ok: true, status: 200, statusText: 'OK', json: async () => ({ count: 2 }) })
      .mockResolvedValueOnce({ ok: true, status: 200, statusText: 'OK', json: async () => ({ success: true }) })
    vi.stubGlobal('fetch', fetchMock)
    const api = await import('./api')
    expect(await api.ExpandDiscographyURL('https://open.spotify.com/artist/x/discography/all')).toEqual(['u1'])
    expect(await api.QueueDiscographyAlbums(['u1'], '/m')).toBe(4)
    expect(await api.DownloadArtistAssets('7', 'Name', '/m')).toBe(2)
    await expect(api.InstallFFmpeg()).resolves.toBeUndefined()
    expect(fetchMock.mock.calls.map((c) => c[0])).toEqual([
      '/api/discography/expand', '/api/discography/queue', '/api/artists/assets', '/api/ffmpeg/install',
    ])
    expect(JSON.parse(fetchMock.mock.calls[1][1].body)).toEqual({ albumUrls: ['u1'], outputDir: '/m' })
    expect(JSON.parse(fetchMock.mock.calls[2][1].body)).toEqual({ artistId: '7', artistName: 'Name', outputDir: '/m' })
  })
})

describe('api.ts source hygiene', () => {
  it('never falls back to console.warn: every web path is a real call or an explicit error', () => {
    expect(apiSource).not.toMatch(/console\.warn\(/)
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

describe('auth token handling in API calls', () => {
  function stubStorage(token?: string) {
    const data = new Map<string, string>(token ? [['flacidal.authToken', token]] : [])
    vi.stubGlobal('localStorage', {
      getItem: (k: string) => data.get(k) ?? null,
      setItem: (k: string, v: string) => void data.set(k, v),
      removeItem: (k: string) => void data.delete(k),
    })
  }
  function stubFetch(status = 200, body: unknown = 'ok') {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: status < 400,
      status,
      statusText: status === 401 ? 'Unauthorized' : 'OK',
      json: async () => body,
    })
    vi.stubGlobal('fetch', fetchMock)
    return fetchMock
  }

  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
    clearWailsRuntime()
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('sends Authorization: Bearer when a token is stored', async () => {
    stubStorage('s3cret')
    const fetchMock = stubFetch(200, { version: '4.0.0' })
    const { GetAppVersion } = await import('./api')
    await GetAppVersion()
    const headers = new Headers(fetchMock.mock.calls[0][1]?.headers)
    expect(headers.get('Authorization')).toBe('Bearer s3cret')
  })

  it('apiPost keeps Content-Type and adds the Bearer header', async () => {
    stubStorage('s3cret')
    const fetchMock = stubFetch(200, true)
    const { SetDownloadFolder } = await import('./api')
    await SetDownloadFolder('/music')
    const headers = new Headers(fetchMock.mock.calls[0][1]?.headers)
    expect(headers.get('Content-Type')).toBe('application/json')
    expect(headers.get('Authorization')).toBe('Bearer s3cret')
  })

  it('sends no Authorization header without a token', async () => {
    stubStorage()
    const fetchMock = stubFetch(200, { version: '4.0.0' })
    const { GetAppVersion } = await import('./api')
    await GetAppVersion()
    const headers = new Headers(fetchMock.mock.calls[0][1]?.headers)
    expect(headers.has('Authorization')).toBe(false)
  })

  it('a 401 notifies listeners and still throws', async () => {
    stubStorage()
    stubFetch(401, { error: 'unauthorized' })
    const { onAuthRequired } = await import('./auth')
    const cb = vi.fn()
    onAuthRequired(cb)
    const { GetAppVersion } = await import('./api')
    await expect(GetAppVersion()).rejects.toThrow('unauthorized')
    expect(cb).toHaveBeenCalledOnce()
  })

  it('ExportFailedDownloads sends the token and reports a 401', async () => {
    stubStorage('s3cret')
    const fetchMock = stubFetch(401, { error: 'unauthorized' })
    const { onAuthRequired } = await import('./auth')
    const cb = vi.fn()
    onAuthRequired(cb)
    const { ExportFailedDownloads } = await import('./api')
    await expect(ExportFailedDownloads('txt')).rejects.toThrow('unauthorized')
    const headers = new Headers(fetchMock.mock.calls[0][1]?.headers)
    expect(headers.get('Authorization')).toBe('Bearer s3cret')
    expect(cb).toHaveBeenCalledOnce()
  })

  it('under Wails there is no fetch and no notification', async () => {
    setWailsRuntime()
    stubStorage('s3cret')
    wailsMock.GetAppVersion.mockResolvedValue('4.0.0')
    const fetchMock = stubFetch(401)
    const { onAuthRequired } = await import('./auth')
    const cb = vi.fn()
    onAuthRequired(cb)
    const { GetAppVersion } = await import('./api')
    await GetAppVersion()
    expect(fetchMock).not.toHaveBeenCalled()
    expect(cb).not.toHaveBeenCalled()
  })
})
