/**
 * Stand-in Wails runtime for the E2E suite.
 *
 * The app pulls its bindings from `../wailsjs/go/main/App.js` and
 * `../wailsjs/runtime/runtime.js`, both of which end up calling into
 * `window.go.main.App.*` and `window.runtime.*` at runtime.
 *
 * None of those globals exist under a plain browser context (the vite dev
 * server), so this file installs them via `page.addInitScript` before unknown
 * app module has a chance to load.
 */
import type { Page } from '@playwright/test'

/** Shape of the window as seen from page.evaluate, after the mocks are installed. */
export interface WinAny {
  go: { main: { App: Record<string, (...args: unknown[]) => Promise<unknown>> } }
  __marker?: number
  __analyzeRes?: unknown
}

export interface WailsOverrides {
  GetConfig?: unknown
  GetDownloadFolder?: string
  IsQueuePaused?: boolean
  GetAppVersion?: string
  DetectSourceFromURL?: unknown
  FetchContentFromURL?: unknown
  ExpandDiscographyURL?: string[]
  GetDownloadHistoryFiltered?: unknown
  ListTrackHistory?: unknown
  GetAvailableSources?: unknown
  GetConversionFormats?: unknown
  IsConverterAvailable?: boolean
  AnalyzeMultiple?: unknown[]
  ListDownloadedFiles?: unknown[]
  ListDownloadedFilesError?: string
  SetDownloadFolderError?: string
  ImportConfigError?: string
  GetSourceHealth?: unknown[]
  Extensions?: unknown[]
  RegistryEntries?: unknown[]
  // Network: stubbed fetch response for /api/* endpoints
  apiAnalyze?: { ok: boolean; body: unknown }
}

export function injectWailsMocks(page: Page, overrides: WailsOverrides = {}) {
  return page.addInitScript((opts) => {
    const defaultConfig = {
      theme: 'dark',
      accentColor: '#f472b6',
      soundEffects: false,
      soundVolume: 70,
      fontFamily: 'Plus Jakarta Sans',
      tidalEnabled: true,
      qobuzEnabled: false,
      qobuzAppId: '',
      qobuzAppSecret: '',
      qobuzAuthToken: '',
      tidalCustomEndpoint: '',
      qobuzCustomEndpoint: '',
      proxyUrl: '',
      skipExisting: true,
      skipUnavailableTracks: false,
      skipDuplicatesByISRC: true,
      autoQualityFallback: true,
      embedLyrics: false,
      generateM3u8: false,
      playlistSubfolder: true,
      outputDir: '/tmp/flacidal-test',
      filenameTemplate: '{artist} - {title}',
      folderTemplate: '{artist}/{album}',
    }

    const config = opts.GetConfig ?? defaultConfig

    // Stateful stand-in for Core's extension management.
    type MockExt = { manifest: Record<string, unknown>; enabled: boolean; hosts: string[]; authSet: Record<string, boolean>; category: string }
    let extensions: MockExt[] = JSON.parse(JSON.stringify(opts.Extensions ?? []))
    let registries = ['https://github.com/kushiemoon-dev/flacidal-extensions']
    const newExtension = (id: string): MockExt => ({
      manifest: { id, name: `Installed ${id}`, version: '1.0.0', author: 'Someone', category: 'download', capabilities: ['source'], permissions: ['network'] },
      enabled: false,
      hosts: ['api.installed.example'],
      authSet: {},
      category: 'download',
    })
    const extensionApp = {
      ListExtensions: async () => JSON.parse(JSON.stringify(extensions)),
      InstallExtension: async (url: string) => {
        if (!url.startsWith('https://')) throw new Error('[extension.insecure_url] extensions are installed from https URLs only')
        const ext = newExtension('from-url')
        extensions = [...extensions, ext]
        return ext.manifest
      },
      InstallExtensionFromRegistry: async (_repo: string, id: string) => {
        const ext = newExtension(id)
        extensions = [...extensions, ext]
        return ext.manifest
      },
      UninstallExtension: async (id: string) => {
        extensions = extensions.filter((e) => e.manifest.id !== id)
      },
      SetExtensionEnabled: async (id: string, enabled: boolean) => {
        extensions = extensions.map((e) => (e.manifest.id === id ? { ...e, enabled } : e))
      },
      SetExtensionAuth: async (id: string, data: Record<string, string>) => {
        extensions = extensions.map((e) =>
          e.manifest.id === id
            ? { ...e, authSet: { ...e.authSet, ...Object.fromEntries(Object.entries(data).filter(([, v]) => v).map(([k]) => [k, true])) } }
            : e,
        )
      },
      GetRegistrySources: async () => [...registries],
      AddRegistrySource: async (url: string) => {
        if (!url.startsWith('https://github.com/')) throw new Error('[extension.registry] registry must be https://github.com/<owner>/<repo>')
        registries = [...registries, url]
      },
      RemoveRegistrySource: async (url: string) => {
        registries = registries.filter((r) => r !== url)
      },
      BrowseRegistry: async (_repo: string) =>
        opts.RegistryEntries ?? [{ id: 'registry-ext', name: 'Registry Ext', description: 'From the registry', category: 'download', latestVersion: '2.0.0', downloadURL: 'https://example.com/r.zip', permissions: ['network'] }],
      GetEffectiveSourceOrder: async () => [
        'tidal', 'qobuz', 'amazon', 'bandcamp',
        ...extensions.filter((e) => e.enabled && e.category === 'download').map((e) => e.manifest.id as string),
        'soulseek',
      ],
    }

    const App = {
      ...extensionApp,
      GetConfig: async () => config,
      SaveConfig: async (_c: unknown) => {},
      ResetToDefaults: async () => defaultConfig,

      GetAppVersion: async () => opts.GetAppVersion ?? '4.0.0-test',

      GetDownloadFolder: async () => opts.GetDownloadFolder ?? '/tmp/flacidal-test',
      ImportConfig: async () => {
        if (opts.ImportConfigError) throw new Error(opts.ImportConfigError)
        return null
      },
      SetDownloadFolder: async (_f: string) => {
        if (opts.SetDownloadFolderError) throw new Error(opts.SetDownloadFolderError)
      },
      SelectDownloadFolder: async () => '/tmp/flacidal-selected',
      OpenDownloadFolder: async (_f: string) => {},
      OpenConfigFolder: async () => {},

      GetAvailableSources: async () =>
        opts.GetAvailableSources ?? [
          { name: 'tidal', displayName: 'Tidal', available: true },
          { name: 'qobuz', displayName: 'Qobuz', available: false },
        ],
      GetPreferredSource: async () => 'tidal',
      SetPreferredSource: async (_s: string) => {},

      DetectSourceFromURL: async (url: string) => {
        if (opts.DetectSourceFromURL !== undefined) return opts.DetectSourceFromURL
        if (!url) return {}
        if (/tidal\.com/i.test(url)) {
          return {
            source: 'tidal',
            displayName: 'Tidal',
            contentType: /album/i.test(url) ? 'album' : 'track',
            available: true,
          }
        }
        if (/qobuz\.com/i.test(url)) {
          return { source: 'qobuz', displayName: 'Qobuz', contentType: 'album', available: false }
        }
        return {}
      },
      FetchContentFromURL: async (_u: string) =>
        opts.FetchContentFromURL ?? {
          type: 'album',
          id: 'test-album-1',
          title: 'Mock Album',
          creator: 'Mock Artist',
          coverUrl: '',
          source: 'tidal',
          tracks: [
            {
              id: 1,
              title: 'Mock Track 1',
              artists: 'Mock Artist',
              duration: 180,
              isrc: 'TEST00000001',
              tidalUrl: 'https://tidal.com/track/1',
              available: true,
              popularity: 50,
              explicit: false,
              previewUrl: '',
            },
          ],
        },
      ExpandDiscographyURL: async (_u: string) =>
        opts.ExpandDiscographyURL ?? [
          'https://open.spotify.com/album/aaa',
          'https://open.spotify.com/album/bbb',
        ],
      ValidateTidalURL: async (_u: string) => ({ valid: true }),
      FetchTidalContent: async (_u: string) => ({
        type: 'album',
        id: 'tid-1',
        title: 'Tidal Album',
        creator: 'Tidal Artist',
        coverUrl: '',
        tracks: [],
      }),
      FetchTidalPlaylist: async (_u: string) => ({}),

      SearchTidal: async (_q: string) => [],
      SearchTidalAlbums: async (_q: string) => [],
      SearchTidalArtists: async (_q: string) => [],

      QueueDownloads: async (..._a: unknown[]) => 0,
      QueueQobuzDownloads: async (..._a: unknown[]) => 0,
      QueueSingleDownload: async (..._a: unknown[]) => {},
      QueueArtistAlbum: async (..._a: unknown[]) => 0,
      DownloadArtistAssets: async (..._a: unknown[]) => 0,
      DownloadTrack: async (..._a: unknown[]) => ({ success: true }),
      CancelDownload: async (_id: number) => {},
      RetryDownload: async (_id: number) => {},
      RetryAllFailed: async () => 0,
      PauseDownloads: async () => true,
      ResumeDownloads: async () => false,
      IsQueuePaused: async () => opts.IsQueuePaused ?? false,
      GetDownloadQueueStatus: async () => ({ items: [] }),
      ExportFailedDownloads: async (_p: string) => '/tmp/failed.txt',

      GetDownloadHistory: async () => [],
      GetDownloadHistoryFiltered: async (_f: unknown) =>
        opts.GetDownloadHistoryFiltered ?? { records: [], total: 0 },
      ListTrackHistory: async (_s: string, _l: number, _o: number) =>
        opts.ListTrackHistory ?? { entries: [], total: 0 },
      DeleteHistoryRecord: async (_id: number) => {},
      ClearDownloadHistory: async () => {},
      RefetchFromHistory: async (_id: string) => ({ url: '' }),

      ListDownloadedFiles: async () => {
        if (opts.ListDownloadedFilesError) throw new Error(opts.ListDownloadedFilesError)
        return opts.ListDownloadedFiles ?? []
      },
      DeleteFile: async (_p: string) => {},
      OpenFLACFilesDialog: async () => [],
      SelectFolderForConversion: async () => [],

      AnalyzeFile: async (_p: string) => ({ verdict: 'lossless' }),
      AnalyzeMultiple: async (_paths: string[]) =>
        opts.AnalyzeMultiple ?? [
          { fileName: 'test.flac', verdict: 'lossless', message: 'Authentic lossless' },
        ],
      QuickAnalyze: async (_p: string) => ({ verdict: 'lossless' }),

      IsConverterAvailable: async () => opts.IsConverterAvailable ?? true,
      GetConversionFormats: async () =>
        opts.GetConversionFormats ?? [
          { name: 'MP3', extension: 'mp3' },
          { name: 'AAC', extension: 'aac' },
          { name: 'ALAC', extension: 'm4a' },
          { name: 'Opus', extension: 'opus' },
          { name: 'Vorbis', extension: 'ogg' },
          { name: 'WAV', extension: 'wav' },
        ],
      ConvertFiles: async (..._a: unknown[]) => [],
      ConvertFolder: async (..._a: unknown[]) => [],
      GetFFmpegInfo: async () => ({ version: '6.0', available: true }),
      GetFFmpegInstallStatus: async () => ({ installed: true }),
      InstallFFmpeg: async () => {},

      FetchAndEmbedLyrics: async (_p: string) => ({ synced: false }),
      FetchAndEmbedLyricsMultiple: async (_p: string[]) => [],
      FetchLyrics: async (..._a: unknown[]) => ({ synced: false }),
      FetchLyricsForFile: async (_p: string) => ({ synced: false }),
      EmbedLyricsToFile: async (..._a: unknown[]) => {},
      GetFileMetadata: async (_p: string) => ({}),
      GetFileCoverArt: async (_p: string) => ({}),

      GetLogs: async () => [],
      GetLogFilePath: async () => '/home/test/.flacidal/flacidal.log',
      ClearLogs: async () => {},
      AddLog: async (_l: string, _m: string) => {},

      GetRecentAlbums: async (_limit: number) => [],

      TestSoulseekConnection: async () => ({ success: false, message: 'mock', username: '' }),
      CheckAPIStatus: async () => [],
      GetSourceHealth: async () => opts.GetSourceHealth ?? [],
      ProbeEndpoints: async () => opts.GetSourceHealth ?? [],
      CheckForUpdate: async () => ({ available: false }),
      GetCacheStats: async () => ({}),
      GetConnectionStatus: async () => ({}),
      GetDownloadOptions: async () => ({}),
      SetDownloadOptions: async (..._a: unknown[]) => {},
      GetMatchFailures: async () => [],
      MatchPlaylistTracks: async (_t: unknown[]) => [],
      MatchSingleTrack: async (_t: unknown) => ({}),
      GetRenameTemplates: async () => [],
      PreviewRename: async (..._a: unknown[]) => [],
      RenameFiles: async (..._a: unknown[]) => [],
      SetTidalCredentials: async (..._a: unknown[]) => {},
      UpdateQobuzCredentials: async (..._a: unknown[]) => {},
      IsQobuzConfigured: async () => false,
      IsDownloaderAvailable: async () => true,
      GetSourceTrack: async (..._a: unknown[]) => ({}),
      GetSourceAlbum: async (..._a: unknown[]) => ({}),
      GetSourcePlaylist: async (..._a: unknown[]) => ({}),
      DownloadTrackFromTidal: async (..._a: unknown[]) => ({}),
      FetchTidalContent_alias: async (..._a: unknown[]) => ({}),
    }

    ;(window as unknown as Record<string, unknown>).go = { main: { App }, app: { App } }

    const noop = () => {}
    const offFn = () => {}
    ;(window as unknown as Record<string, unknown>).runtime = {
      EventsOn: (_n: string, _cb: unknown) => offFn,
      EventsOff: noop,
      EventsEmit: noop,
      EventsOnce: (_n: string, _cb: unknown) => offFn,
      EventsOnMultiple: (_n: string, _cb: unknown, _max: number) => offFn,
      LogPrint: noop,
      LogTrace: noop,
      LogDebug: noop,
      LogInfo: noop,
      LogWarning: noop,
      LogError: noop,
      LogFatal: noop,
      BrowserOpenURL: noop,
      WindowReload: noop,
      WindowReloadApp: noop,
      WindowSetSystemDefaultTheme: noop,
      WindowSetLightTheme: noop,
      WindowSetDarkTheme: noop,
      WindowCenter: noop,
      WindowSetTitle: noop,
      WindowFullscreen: noop,
      WindowUnfullscreen: noop,
      WindowIsFullscreen: () => false,
      WindowGetSize: () => ({ w: 1280, h: 800 }),
      WindowSetSize: noop,
      WindowSetMinSize: noop,
      WindowSetMaxSize: noop,
      WindowSetAlwaysOnTop: noop,
      WindowSetPosition: noop,
      WindowGetPosition: () => ({ x: 0, y: 0 }),
      WindowHide: noop,
      WindowShow: noop,
      WindowMaximise: noop,
      WindowUnmaximise: noop,
      WindowToggleMaximise: noop,
      WindowIsMaximised: () => false,
      WindowMinimise: noop,
      WindowUnminimise: noop,
      WindowIsMinimised: () => false,
      WindowIsNormal: () => true,
      WindowSetBackgroundColour: noop,
      ScreenGetAll: async () => [],
      ClipboardGetText: async () => '',
      ClipboardSetText: async (_t: string) => true,
      Quit: noop,
      Hide: noop,
      Show: noop,
      Environment: async () => ({ buildType: 'dev', platform: 'linux', arch: 'amd64' }),
      OnFileDrop: (_cb: unknown, _useDropTarget: boolean) => {},
      OnFileDropOff: () => {},
    }

    // Flag Wails as ready, in case some code path waits on this
    ;(window as unknown as Record<string, unknown>).WailsInvoke = noop
  }, overrides)
}
