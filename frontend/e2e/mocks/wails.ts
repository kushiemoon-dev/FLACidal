/**
 * Wails runtime mock for E2E tests.
 *
 * The frontend imports from `../wailsjs/go/main/App.js` and
 * `../wailsjs/runtime/runtime.js`, which at runtime call into
 * `window.go.main.App.*` and `window.runtime.*`.
 *
 * In a browser context (vite dev server) those globals do not exist, so we
 * inject mocks via `page.addInitScript` BEFORE any module loads.
 */
import type { Page } from '@playwright/test'

export interface WailsOverrides {
  GetConfig?: any
  GetDownloadFolder?: string
  IsQueuePaused?: boolean
  GetAppVersion?: string
  DetectSourceFromURL?: any
  FetchContentFromURL?: any
  ExpandDiscographyURL?: string[]
  GetDownloadHistoryFiltered?: any
  GetAvailableSources?: any
  GetConversionFormats?: any
  IsConverterAvailable?: boolean
  AnalyzeMultiple?: any[]
  ListDownloadedFiles?: any[]
  // Network: mock fetch for /api/* endpoints
  apiAnalyze?: { ok: boolean; body: any }
}

export function injectWailsMocks(page: Page, overrides: WailsOverrides = {}) {
  return page.addInitScript((opts) => {
    // ---------- Default config ---------- //
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

    // ---------- App methods ---------- //
    const App = {
      // Config
      GetConfig: async () => config,
      SaveConfig: async (_c: any) => {},
      ResetToDefaults: async () => defaultConfig,

      // App / version
      GetAppVersion: async () => opts.GetAppVersion ?? '4.0.0-test',

      // Folder
      GetDownloadFolder: async () => opts.GetDownloadFolder ?? '/tmp/flacidal-test',
      SetDownloadFolder: async (_f: string) => {},
      SelectDownloadFolder: async () => '/tmp/flacidal-selected',
      OpenDownloadFolder: async (_f: string) => {},
      OpenConfigFolder: async () => {},

      // Sources
      GetAvailableSources: async () =>
        opts.GetAvailableSources ?? [
          { name: 'tidal', displayName: 'Tidal', available: true },
          { name: 'qobuz', displayName: 'Qobuz', available: false },
        ],
      GetPreferredSource: async () => 'tidal',
      SetPreferredSource: async (_s: string) => {},

      // Source detection / fetch
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

      // Search
      SearchTidal: async (_q: string) => [],
      SearchTidalAlbums: async (_q: string) => [],
      SearchTidalArtists: async (_q: string) => [],

      // Download / queue
      QueueDownloads: async (..._a: any[]) => 0,
      QueueQobuzDownloads: async (..._a: any[]) => 0,
      QueueSingleDownload: async (..._a: any[]) => {},
      QueueArtistAlbum: async (..._a: any[]) => 0,
      DownloadArtistAssets: async (..._a: any[]) => 0,
      DownloadTrack: async (..._a: any[]) => ({ success: true }),
      CancelDownload: async (_id: number) => {},
      RetryDownload: async (_id: number) => {},
      RetryAllFailed: async () => 0,
      PauseDownloads: async () => true,
      ResumeDownloads: async () => false,
      IsQueuePaused: async () => opts.IsQueuePaused ?? false,
      GetDownloadQueueStatus: async () => ({ items: [] }),
      ExportFailedDownloads: async (_p: string) => '/tmp/failed.txt',

      // History
      GetDownloadHistory: async () => [],
      GetDownloadHistoryFiltered: async (_f: any) =>
        opts.GetDownloadHistoryFiltered ?? { records: [], total: 0 },
      DeleteHistoryRecord: async (_id: number) => {},
      ClearDownloadHistory: async () => {},
      RefetchFromHistory: async (_id: string) => ({ url: '' }),

      // Files
      ListDownloadedFiles: async () => opts.ListDownloadedFiles ?? [],
      DeleteFile: async (_p: string) => {},
      OpenFLACFilesDialog: async () => [],
      SelectFolderForConversion: async () => [],

      // Analyzer
      AnalyzeFile: async (_p: string) => ({ verdict: 'lossless' }),
      AnalyzeMultiple: async (_paths: string[]) =>
        opts.AnalyzeMultiple ?? [
          { fileName: 'test.flac', verdict: 'lossless', message: 'Authentic lossless' },
        ],
      QuickAnalyze: async (_p: string) => ({ verdict: 'lossless' }),

      // Conversion
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
      ConvertFiles: async (..._a: any[]) => [],
      ConvertFolder: async (..._a: any[]) => [],
      GetFFmpegInfo: async () => ({ version: '6.0', available: true }),
      GetFFmpegInstallStatus: async () => ({ installed: true }),
      InstallFFmpeg: async () => {},

      // Lyrics / metadata
      FetchAndEmbedLyrics: async (_p: string) => ({ synced: false }),
      FetchAndEmbedLyricsMultiple: async (_p: string[]) => [],
      FetchLyrics: async (..._a: any[]) => ({ synced: false }),
      FetchLyricsForFile: async (_p: string) => ({ synced: false }),
      EmbedLyricsToFile: async (..._a: any[]) => {},
      GetFileMetadata: async (_p: string) => ({}),
      GetFileCoverArt: async (_p: string) => ({}),

      // Logs
      GetLogs: async () => [],
      ClearLogs: async () => {},
      AddLog: async (_l: string, _m: string) => {},

      // Recent albums (home page) + Soulseek status
      GetRecentAlbums: async (_limit: number) => [],
      GetSldlStatus: async () => ({ installed: false, version: '' }),
      InstallSldl: async () => {},
      GetSldlBinaryPath: async () => '',

      // Misc
      CheckAPIStatus: async () => [],
      CheckForUpdate: async () => ({ available: false }),
      GetCacheStats: async () => ({}),
      GetConnectionStatus: async () => ({}),
      GetDownloadOptions: async () => ({}),
      SetDownloadOptions: async (..._a: any[]) => {},
      GetMatchFailures: async () => [],
      MatchPlaylistTracks: async (_t: any[]) => [],
      MatchSingleTrack: async (_t: any) => ({}),
      GetRenameTemplates: async () => [],
      PreviewRename: async (..._a: any[]) => [],
      RenameFiles: async (..._a: any[]) => [],
      SetTidalCredentials: async (..._a: any[]) => {},
      UpdateQobuzCredentials: async (..._a: any[]) => {},
      IsQobuzConfigured: async () => false,
      IsDownloaderAvailable: async () => true,
      GetSourceTrack: async (..._a: any[]) => ({}),
      GetSourceAlbum: async (..._a: any[]) => ({}),
      GetSourcePlaylist: async (..._a: any[]) => ({}),
      DownloadTrackFromTidal: async (..._a: any[]) => ({}),
      FetchTidalContent_alias: async (..._a: any[]) => ({}),
    }

    ;(window as any).go = { main: { App } }

    // ---------- Wails runtime ---------- //
    const noop = () => {}
    const offFn = () => {}
    ;(window as any).runtime = {
      EventsOn: (_n: string, _cb: any) => offFn,
      EventsOff: noop,
      EventsEmit: noop,
      EventsOnce: (_n: string, _cb: any) => offFn,
      EventsOnMultiple: (_n: string, _cb: any, _max: number) => offFn,
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
      OnFileDrop: (_cb: any, _useDropTarget: boolean) => {},
      OnFileDropOff: () => {},
    }

    // Mark wails as ready in case any code awaits this
    ;(window as any).WailsInvoke = noop
  }, overrides as any)
}

/**
 * Mock the HTTP /api/analyze endpoint via fetch interception.
 * Used by the (currently unrouted) Analyzer.svelte page if it ever gets wired up.
 */
export async function mockApiAnalyze(page: Page, body: any, status = 200) {
  await page.route('**/api/analyze', (route) =>
    route.fulfill({
      status,
      contentType: 'application/json',
      body: JSON.stringify(body),
    }),
  )
}
