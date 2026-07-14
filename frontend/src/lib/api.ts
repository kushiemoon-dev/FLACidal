// Runtime-detecting API layer.
//
// FLACidal's frontend bundle runs in two contexts:
//  - the Wails desktop webview, where backend calls are Go bindings exposed
//    on window.go.app.App.* (see ../../wailsjs/go/app/App.js)
//  - a plain browser pointed at the headless HTTP server (internal/api/),
//    where the same calls must go over fetch() to /api/*
//
// Every exported function here picks the right transport at call time via
// isWailsRuntime() and returns data shaped the same way regardless of which
// backend answered, so the 18 consumer components don't need to know or
// care which mode they're running in. Import from here instead of
// 'wailsjs/go/app/App.js' directly.
//
// Route mapping and known gaps between the two backends are documented next
// to each function below (also summarized in the migration report).

import * as Wails from '../../wailsjs/go/app/App.js'

// ---------------------------------------------------------------------------
// Runtime detection
// ---------------------------------------------------------------------------

let cachedIsWails: boolean | null = null

/** Detects whether we're running inside the Wails desktop webview (result is cached). */
export function isWailsRuntime(): boolean {
  if (cachedIsWails === null) {
    const w = window as any
    cachedIsWails = typeof window !== 'undefined' && !!w.go?.app?.App && !!w.runtime
  }
  return cachedIsWails
}

// ---------------------------------------------------------------------------
// REST transport helpers
// ---------------------------------------------------------------------------

const API_BASE = '/api'

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, init)
  if (!res.ok) {
    let message = `${res.status} ${res.statusText}`
    try {
      const body = await res.json()
      if (body?.error) message = body.error
    } catch {
      // response wasn't JSON — keep the status-based message
    }
    throw new Error(message)
  }
  return res.json() as Promise<T>
}

function apiGet<T>(path: string): Promise<T> {
  return apiFetch<T>(path)
}

function apiPost<T>(path: string, body?: unknown): Promise<T> {
  return apiFetch<T>(path, {
    method: 'POST',
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
}

function apiDelete<T>(path: string): Promise<T> {
  return apiFetch<T>(path, { method: 'DELETE' })
}

/** Builds a `?a=1&b=2` query string, skipping undefined/empty values. */
function qs(params: Record<string, string | number | boolean | undefined>): string {
  const entries = Object.entries(params).filter(([, v]) => v !== undefined && v !== '')
  if (entries.length === 0) return ''
  return '?' + entries.map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`).join('&')
}

// ---------------------------------------------------------------------------
// Loose response types
//
// The codebase's existing Wails call sites already lean on `any` / optional
// chaining rather than the generated wailsjs classes, so these stay
// intentionally permissive (extra `[key: string]: any`) instead of
// re-declaring every backend struct field — just enough shape for the
// fields components actually read (per the call-site audit).
// ---------------------------------------------------------------------------

export interface AnalysisResult {
  filePath: string
  fileName: string
  isTrueLossless: boolean
  confidence: number
  spectrumCutoff: number
  expectedCutoff: number
  verdict: string
  verdictLabel: string
  details: string
  sampleRate: number
  bitDepth: number
}

export interface ConversionResult {
  sourcePath: string
  outputPath: string
  success: boolean
  error?: string
  outputSize?: number
  sourceSize?: number
}

export interface ConversionFormat {
  id: string
  name: string
  extension: string
  qualities: string[]
  description: string
}

export interface DownloadedFileInfo {
  path: string
  name: string
  size: number
  modTime: string
  title: string
  artist: string
  album: string
  [key: string]: any
}

export interface RenamePreview {
  oldPath: string
  oldName: string
  newName: string
  newPath: string
  hasError: boolean
  error?: string
}

export interface RenameResult {
  oldPath: string
  newPath: string
  success: boolean
  error?: string
}

export interface LogEntry {
  timestamp: string
  level: string
  message: string
}

// ---------------------------------------------------------------------------
// Analysis — POST /api/analyze/multiple
// ---------------------------------------------------------------------------

export async function AnalyzeMultiple(paths: string[]): Promise<AnalysisResult[]> {
  if (isWailsRuntime()) {
    return Wails.AnalyzeMultiple(paths) as unknown as Promise<AnalysisResult[]>
  }

  const raw = await apiPost<any[]>('/analyze/multiple', { paths })
  // The REST endpoint's response shape (isUpscaled/spectralCutoff/message)
  // deliberately differs from core.AnalysisResult (isTrueLossless/
  // spectrumCutoff/details) — normalize here so components see one
  // consistent shape either way.
  // Known gaps: the REST endpoint doesn't return filePath or expectedCutoff.
  // filePath is reconstructed from the request's path order (the server
  // preserves input order); expectedCutoff defaults to 0.
  return raw.map((r, i) => ({
    filePath: paths[i] ?? '',
    fileName: r.fileName,
    isTrueLossless: !r.isUpscaled,
    confidence: r.confidence,
    spectrumCutoff: r.spectralCutoff,
    expectedCutoff: 0,
    verdict: r.verdict,
    verdictLabel: r.verdictLabel,
    details: r.message,
    sampleRate: r.sampleRate,
    bitDepth: r.bitDepth,
  }))
}

// ---------------------------------------------------------------------------
// Downloads / Queue
// ---------------------------------------------------------------------------

export async function CancelDownload(trackId: number): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.CancelDownload(trackId)
  }
  await apiPost(`/downloads/cancel/${trackId}`)
}

// Note: Wails' Pause/ResumeDownloads return whether the operation actually
// changed state; the REST endpoints always report a fixed paused:true/false
// regardless of prior state. No current call site reads this return value
// (they set their local store optimistically), so the difference is
// harmless today — flagged here in case a future caller relies on it.
export async function PauseDownloads(): Promise<boolean> {
  if (isWailsRuntime()) {
    return Wails.PauseDownloads()
  }
  const { paused } = await apiPost<{ paused: boolean }>('/downloads/pause')
  return paused
}

export async function ResumeDownloads(): Promise<boolean> {
  if (isWailsRuntime()) {
    return Wails.ResumeDownloads()
  }
  const { paused } = await apiPost<{ paused: boolean }>('/downloads/resume')
  return !paused
}

export async function IsQueuePaused(): Promise<boolean> {
  if (isWailsRuntime()) {
    return Wails.IsQueuePaused()
  }
  const { paused } = await apiGet<{ paused: boolean }>('/downloads/paused')
  return paused
}

export async function RetryAllFailed(): Promise<number> {
  if (isWailsRuntime()) {
    return Wails.RetryAllFailed()
  }
  const { retried } = await apiPost<{ retried: number }>('/downloads/retry-all')
  return retried
}

/**
 * Exports failed downloads as a TXT or CSV file.
 * Wails: opens a native "Save As" dialog and returns the saved path.
 * Browser: no native dialog exists, so the file is fetched and pushed
 * through the browser's own download flow (temporary `<a download>` click).
 * Returns '' in browser mode since there's no server-side path to report —
 * mirrors Wails' own '' return when the user cancels its dialog.
 */
export async function ExportFailedDownloads(format: 'txt' | 'csv'): Promise<string> {
  if (isWailsRuntime()) {
    return Wails.ExportFailedDownloads(format)
  }

  const res = await fetch(`${API_BASE}/downloads/export?format=${encodeURIComponent(format)}`)
  if (!res.ok) {
    const body = await res.json().catch(() => null)
    throw new Error(body?.error || `${res.status} ${res.statusText}`)
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `failed_downloads.${format}`
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
  return ''
}

export async function QueueDownloads(
  tracks: any[],
  outputDir: string,
  contentName: string,
  contentId: string,
  contentType: string
): Promise<number> {
  if (isWailsRuntime()) {
    return Wails.QueueDownloads(tracks as any, outputDir, contentName, contentId, contentType)
  }
  // Known gap: unlike the Wails path, the REST endpoint doesn't yet persist
  // a content-level DownloadRecord for contentId/contentType, so
  // playlist/album progress in History won't populate for downloads queued
  // through the headless server. See migration report.
  const { queued } = await apiPost<{ queued: number }>('/downloads/queue', { tracks, outputDir, contentName })
  return queued
}

export async function QueueSingleDownload(
  trackId: number,
  outputDir: string,
  title: string,
  artist: string
): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.QueueSingleDownload(trackId, outputDir, title, artist)
  }
  await apiPost('/downloads/single', { trackId, outputDir, title, artist })
}

export async function QueueArtistAlbum(albumId: string, artistName: string, outputDir: string): Promise<number> {
  if (isWailsRuntime()) {
    return Wails.QueueArtistAlbum(albumId, artistName, outputDir)
  }
  const { queued } = await apiPost<{ queued: number }>('/downloads/queue/album', { albumId, artistName, outputDir })
  return queued
}

// ---------------------------------------------------------------------------
// History
// ---------------------------------------------------------------------------

// Return type is intentionally `any` for `records`: consumers (e.g.
// History.svelte) declare their own local DownloadRecord interface, and
// TypeScript won't structurally reconcile two independently-declared
// same-named interfaces even with an index signature present.
export async function GetDownloadHistoryFiltered(
  filter: Record<string, any>
): Promise<{ records: any[]; total: number }> {
  if (isWailsRuntime()) {
    return Wails.GetDownloadHistoryFiltered(filter) as unknown as Promise<{ records: any[]; total: number }>
  }
  const query = qs({
    limit: filter.limit,
    offset: filter.offset,
    contentType: filter.contentType,
    search: filter.search,
  })
  return apiGet(`/history/filtered${query}`)
}

export async function DeleteHistoryRecord(id: number): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.DeleteHistoryRecord(id)
  }
  await apiDelete(`/history/${id}`)
}

export async function ClearDownloadHistory(): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.ClearDownloadHistory()
  }
  await apiPost('/history/clear')
}

export async function RefetchFromHistory(tidalContentId: string): Promise<any> {
  if (isWailsRuntime()) {
    return Wails.RefetchFromHistory(tidalContentId)
  }
  return apiPost(`/history/refetch/${encodeURIComponent(tidalContentId)}`)
}

// ---------------------------------------------------------------------------
// Files
// ---------------------------------------------------------------------------

export async function ListDownloadedFiles(): Promise<DownloadedFileInfo[]> {
  if (isWailsRuntime()) {
    return Wails.ListDownloadedFiles() as unknown as Promise<DownloadedFileInfo[]>
  }
  return apiGet('/files')
}

export async function DeleteFile(path: string): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.DeleteFile(path)
  }
  await apiDelete(`/files?path=${encodeURIComponent(path)}`)
}

// Return type is intentionally `any`: MetadataModal.svelte declares its own
// local FLACMetadata interface, and TypeScript won't structurally reconcile
// two independently-declared same-named interfaces (see GetDownloadHistoryFiltered above).
export async function GetFileMetadata(filePath: string): Promise<any> {
  if (isWailsRuntime()) {
    return Wails.GetFileMetadata(filePath)
  }
  return apiGet(`/files/metadata?path=${encodeURIComponent(filePath)}`)
}

export async function GetFileCoverArt(filePath: string): Promise<{ data: string; mimeType: string }> {
  if (isWailsRuntime()) {
    return Wails.GetFileCoverArt(filePath) as unknown as Promise<{ data: string; mimeType: string }>
  }
  return apiGet(`/files/cover?path=${encodeURIComponent(filePath)}`)
}

export async function GetRenameTemplates(): Promise<Array<{ name: string; template: string }>> {
  if (isWailsRuntime()) {
    return Wails.GetRenameTemplates() as unknown as Promise<Array<{ name: string; template: string }>>
  }
  return apiGet('/files/templates')
}

export async function PreviewRename(files: string[], template: string): Promise<RenamePreview[]> {
  if (isWailsRuntime()) {
    return Wails.PreviewRename(files, template)
  }
  return apiPost('/files/rename/preview', { files, template })
}

export async function RenameFiles(files: string[], template: string): Promise<RenameResult[]> {
  if (isWailsRuntime()) {
    return Wails.RenameFiles(files, template)
  }
  return apiPost('/files/rename', { files, template })
}

// ---------------------------------------------------------------------------
// Conversion
// ---------------------------------------------------------------------------

export async function ConvertFiles(
  files: string[],
  format: string,
  quality: string,
  outputDir: string,
  deleteSource: boolean
): Promise<ConversionResult[]> {
  if (isWailsRuntime()) {
    return Wails.ConvertFiles(files, format, quality, outputDir, deleteSource)
  }
  return apiPost('/convert', { files, format, quality, outputDir, deleteSource })
}

export async function GetConversionFormats(): Promise<ConversionFormat[]> {
  if (isWailsRuntime()) {
    return Wails.GetConversionFormats()
  }
  return apiGet('/convert/formats')
}

export async function GetFFmpegInfo(): Promise<any> {
  if (isWailsRuntime()) {
    return Wails.GetFFmpegInfo()
  }
  return apiGet('/convert/ffmpeg')
}

export async function IsConverterAvailable(): Promise<boolean> {
  if (isWailsRuntime()) {
    return Wails.IsConverterAvailable()
  }
  const { available } = await apiGet<{ available: boolean }>('/convert/available')
  return available
}

// ---------------------------------------------------------------------------
// Config / App info / Folder
// ---------------------------------------------------------------------------

export async function GetConfig(): Promise<any> {
  if (isWailsRuntime()) {
    return Wails.GetConfig()
  }
  return apiGet('/config')
}

export async function GetAppVersion(): Promise<string> {
  if (isWailsRuntime()) {
    return Wails.GetAppVersion()
  }
  // Known gap: the REST server currently reports a hardcoded "1.0.0"
  // instead of the real build version (see migration report).
  const { version } = await apiGet<{ version: string }>('/version')
  return version
}

export async function GetDownloadFolder(): Promise<string> {
  if (isWailsRuntime()) {
    return Wails.GetDownloadFolder()
  }
  const { folder } = await apiGet<{ folder: string }>('/folder')
  return folder
}

// ---------------------------------------------------------------------------
// Logs
//
// Known gap: the headless server's /api/logs and /api/logs/clear are stubs
// (no server-side log buffer wired up yet, unlike the Wails app's
// logBuffer) — GetLogs() always resolves to [] in browser mode and
// ClearLogs() is a no-op. Pre-existing backend limitation, out of scope for
// this migration. See migration report.
// ---------------------------------------------------------------------------

export async function GetLogs(): Promise<LogEntry[]> {
  if (isWailsRuntime()) {
    return Wails.GetLogs()
  }
  return apiGet('/logs')
}

export async function ClearLogs(): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.ClearLogs()
  }
  await apiPost('/logs/clear')
}

// ---------------------------------------------------------------------------
// Content / Search
// ---------------------------------------------------------------------------

export async function FetchContentFromURL(url: string): Promise<any> {
  if (isWailsRuntime()) {
    return Wails.FetchContentFromURL(url)
  }
  return apiPost('/content/fetch', { url })
}

export async function SearchTidal(query: string): Promise<any[]> {
  if (isWailsRuntime()) {
    return Wails.SearchTidal(query)
  }
  return apiGet(`/content/search${qs({ q: query })}`)
}

export async function SearchTidalAlbums(query: string): Promise<any[]> {
  if (isWailsRuntime()) {
    return Wails.SearchTidalAlbums(query)
  }
  return apiGet(`/content/search/albums${qs({ q: query })}`)
}

export async function SearchTidalArtists(query: string): Promise<any[]> {
  if (isWailsRuntime()) {
    return Wails.SearchTidalArtists(query)
  }
  return apiGet(`/content/search/artists${qs({ q: query })}`)
}

export async function SearchDeezer(query: string): Promise<any[]> {
  if (isWailsRuntime()) {
    return Wails.SearchDeezer(query)
  }
  return apiGet(`/content/search/deezer${qs({ q: query })}`)
}

// ---------------------------------------------------------------------------
// Lyrics
// ---------------------------------------------------------------------------

export async function FetchAndEmbedLyricsMultiple(filePaths: string[]): Promise<any[]> {
  if (isWailsRuntime()) {
    return Wails.FetchAndEmbedLyricsMultiple(filePaths)
  }
  return apiPost('/lyrics/fetch-embed/multiple', { filePaths })
}

// ---------------------------------------------------------------------------
// Native OS dialogs — no browser equivalent
//
// These four Wails calls open native OS dialogs (file picker / folder
// picker) or the system file manager. A browser sandbox has no access to
// real filesystem paths (an <input type="file"> only ever exposes a
// filename, never an absolute path) or to the local file manager, and the
// server may not even be running on the same machine as the browser — so
// there's no way to honestly fulfill these in browser mode.
//
// Every call site for these four functions already treats a falsy/empty
// result as "user cancelled" and no-ops (confirmed by call-site audit), so
// resolving to that same "cancelled" value in browser mode degrades
// cleanly with zero component changes and zero crashes.
// ---------------------------------------------------------------------------

export async function OpenFLACFilesDialog(): Promise<string[]> {
  if (isWailsRuntime()) {
    return Wails.OpenFLACFilesDialog()
  }
  console.warn('OpenFLACFilesDialog: not available in browser mode (no server-side file path can be obtained from a browser file picker)')
  return []
}

export async function SelectDownloadFolder(): Promise<string> {
  if (isWailsRuntime()) {
    return Wails.SelectDownloadFolder()
  }
  console.warn('SelectDownloadFolder: not available in browser mode (no native folder picker in a browser context)')
  return ''
}

export async function SelectFolderForConversion(): Promise<string[]> {
  if (isWailsRuntime()) {
    return Wails.SelectFolderForConversion()
  }
  console.warn('SelectFolderForConversion: not available in browser mode (same limitation as OpenFLACFilesDialog)')
  return []
}

export async function OpenDownloadFolder(path: string): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.OpenDownloadFolder(path)
  }
  console.warn('OpenDownloadFolder: not available in browser mode (no access to the local file manager from a web page)')
}

// ---------------------------------------------------------------------------
// Config (additional — Home.svelte / Settings.svelte)
// ---------------------------------------------------------------------------

export async function SetDownloadFolder(folder: string): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.SetDownloadFolder(folder)
  }
  await apiPost('/folder', { folder })
}

/**
 * Known gap: unlike the Wails path, the REST handler only persists the
 * config — it doesn't re-apply live settings to the download manager, the
 * downloader's proxy/quality options, or re-initialize the Soulseek source.
 * Some settings may need a server restart to take effect in browser mode.
 * See migration report.
 */
export async function SaveConfig(config: any): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.SaveConfig(config)
  }
  await apiPost('/config', config)
}

export async function GetDownloadOptions(): Promise<any> {
  if (isWailsRuntime()) {
    return Wails.GetDownloadOptions()
  }
  return apiGet('/downloads/options')
}

export async function SetDownloadOptions(
  quality: string,
  fileNameFormat: string,
  organizeFolders: boolean,
  embedCover: boolean,
  saveCoverFile: boolean,
  autoAnalyze: boolean
): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.SetDownloadOptions(quality, fileNameFormat, organizeFolders, embedCover, saveCoverFile, autoAnalyze)
  }
  await apiPost('/downloads/options', { quality, fileNameFormat, organizeFolders, embedCover, saveCoverFile, autoAnalyze })
}

export async function ResetToDefaults(): Promise<any> {
  if (isWailsRuntime()) {
    return Wails.ResetToDefaults()
  }
  return apiPost('/config/reset')
}

export async function OpenConfigFolder(): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.OpenConfigFolder()
  }
  console.warn('OpenConfigFolder: not available in browser mode (no access to the local file manager from a web page)')
}

// ---------------------------------------------------------------------------
// Sources (additional)
// ---------------------------------------------------------------------------

export async function DetectSourceFromURL(url: string): Promise<any> {
  if (isWailsRuntime()) {
    return Wails.DetectSourceFromURL(url)
  }
  const result = await apiPost<any>('/sources/detect', { url })
  // Normalize: the REST failure branch omits contentType/id (Wails includes
  // them as empty strings) — fill them in so callers can rely on both keys.
  return { contentType: '', id: '', ...result }
}

export async function SetSourceOrder(order: string[]): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.SetSourceOrder(order)
  }
  await apiPost('/sources/order', { order })
}

export async function GetSldlStatus(): Promise<any> {
  if (isWailsRuntime()) {
    return Wails.GetSldlStatus()
  }
  return apiGet('/sources/soulseek/status')
}

export async function TestSoulseekConnection(username: string, password: string): Promise<any> {
  if (isWailsRuntime()) {
    return Wails.TestSoulseekConnection(username, password)
  }
  return apiPost('/sources/soulseek/test', { username, password })
}

// ---------------------------------------------------------------------------
// Content (additional — Tidal-specific Wails calls mapped onto the generic,
// multi-source REST endpoints already used by FetchContentFromURL/etc.)
// ---------------------------------------------------------------------------

/**
 * Known gap: the REST /content/fetch handler only supports track/album/
 * playlist via the generic multi-source sourceManager — Wails'
 * FetchTidalContent additionally supports Tidal "mix" and "artist" URLs,
 * which will fail (400/500) here. See migration report.
 */
export async function FetchTidalContent(url: string): Promise<any> {
  if (isWailsRuntime()) {
    return Wails.FetchTidalContent(url)
  }
  return apiPost('/content/fetch', { url })
}

/**
 * Known gap: REST validates against any registered source, not exclusively
 * Tidal, and its success shape uses `contentType` instead of `type` —
 * normalized here to match what Wails returns.
 */
export async function ValidateTidalURL(url: string): Promise<any> {
  if (isWailsRuntime()) {
    return Wails.ValidateTidalURL(url)
  }
  const result = await apiPost<any>('/content/validate', { url })
  if (result?.valid && result.contentType !== undefined) {
    return { ...result, type: result.contentType }
  }
  return result
}

export async function QueueQobuzDownloads(tracks: any[], outputDir: string, contentName: string): Promise<number> {
  if (isWailsRuntime()) {
    return Wails.QueueQobuzDownloads(tracks as any, outputDir, contentName)
  }
  const { queued } = await apiPost<{ queued: number }>('/downloads/queue/qobuz', { tracks, outputDir, contentName })
  return queued
}

export async function GetRecentAlbums(limit: number): Promise<any[]> {
  if (isWailsRuntime()) {
    return Wails.GetRecentAlbums(limit)
  }
  return apiGet(`/history/recent${qs({ limit })}`)
}

// ---------------------------------------------------------------------------
// System (additional)
// ---------------------------------------------------------------------------

/**
 * No REST route: this is a stateless, side-effect-free public GitHub API
 * call, so browser mode just makes it directly (the same approach
 * About.svelte already uses for repo stats) instead of round-tripping
 * through the server.
 */
export async function CheckForUpdate(): Promise<{ hasUpdate: boolean; version: string; url: string; releaseUrl: string }> {
  if (isWailsRuntime()) {
    return Wails.CheckForUpdate() as unknown as Promise<{ hasUpdate: boolean; version: string; url: string; releaseUrl: string }>
  }
  try {
    const res = await fetch('https://api.github.com/repos/kushiemoon-dev/flacidal/releases/latest', {
      headers: { Accept: 'application/vnd.github.v3+json' },
    })
    if (!res.ok) {
      return { hasUpdate: false, version: '', url: '', releaseUrl: '' }
    }
    const release = await res.json()
    const latestVersion = String(release.tag_name || '').replace(/^v/, '')
    const currentVersion = await GetAppVersion()
    const hasUpdate = latestVersion !== '' && latestVersion !== currentVersion && latestVersion > currentVersion
    const downloadUrl = release.assets?.[0]?.browser_download_url || release.html_url
    return { hasUpdate, version: latestVersion, url: downloadUrl, releaseUrl: release.html_url }
  } catch {
    return { hasUpdate: false, version: '', url: '', releaseUrl: '' }
  }
}

// ---------------------------------------------------------------------------
// Deep structural gaps — Wails mode works unchanged; browser mode throws a
// clear, catchable error instead of crashing on an undefined window.go
// binding or silently no-op'ing an action the user explicitly triggered
// (Install buttons, discography queueing, etc — unlike the native-OS
// dialogs above, an empty/falsy result here could be misread as a real
// negative finding rather than "not implemented", so these throw instead).
//
// Implementing these server-side would each require new backend subsystems
// that don't exist on the headless server today: a Spotify search client
// (ExpandDiscographyURL/QueueDiscographyAlbums), a Tidal artist-image client
// (DownloadArtistAssets), Amazon/Soulseek source wiring (CheckAPIStatus/
// GetSourceHealth), or an install-progress relay (InstallFFmpeg/InstallSldl).
// See migration report for the full reasoning per function.
// ---------------------------------------------------------------------------

function unavailableInBrowser(name: string, reason: string): never {
  throw new Error(`${name} isn't available in browser mode — ${reason}`)
}

export async function DownloadArtistAssets(artistId: string, artistName: string, outputDir: string): Promise<number> {
  if (isWailsRuntime()) {
    return Wails.DownloadArtistAssets(artistId, artistName, outputDir)
  }
  return unavailableInBrowser('DownloadArtistAssets', 'the headless server has no Tidal artist-image client wired up yet')
}

export async function ExpandDiscographyURL(url: string): Promise<string[]> {
  if (isWailsRuntime()) {
    return Wails.ExpandDiscographyURL(url)
  }
  return unavailableInBrowser('ExpandDiscographyURL', 'the headless server has no Spotify search client wired up yet')
}

export async function QueueDiscographyAlbums(albumUrls: string[], outputDir: string): Promise<number> {
  if (isWailsRuntime()) {
    return Wails.QueueDiscographyAlbums(albumUrls, outputDir)
  }
  return unavailableInBrowser('QueueDiscographyAlbums', 'the headless server has no Spotify search client wired up yet')
}

export async function CheckAPIStatus(): Promise<any[]> {
  if (isWailsRuntime()) {
    return Wails.CheckAPIStatus()
  }
  return unavailableInBrowser('CheckAPIStatus', 'not yet implemented on the headless server')
}

export async function GetSourceHealth(): Promise<any[]> {
  if (isWailsRuntime()) {
    return Wails.GetSourceHealth()
  }
  return unavailableInBrowser('GetSourceHealth', 'not yet implemented on the headless server (needs Amazon/Soulseek source wiring)')
}

export async function InstallFFmpeg(): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.InstallFFmpeg()
  }
  return unavailableInBrowser('InstallFFmpeg', 'installing binaries onto a possibly-remote server needs progress-reporting infrastructure that does not exist yet')
}

export async function InstallSldl(): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.InstallSldl()
  }
  return unavailableInBrowser('InstallSldl', 'installing binaries onto a possibly-remote server needs progress-reporting infrastructure that does not exist yet')
}
