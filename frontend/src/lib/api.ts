// API layer that detects its runtime environment.
//
// This bundle ships into two different environments:
//  - inside the Wails desktop webview, calls reach the backend through Go
//    bindings at window.go.app.App.* (see ../../wailsjs/go/app/App.js)
//  - inside a plain browser talking to the headless HTTP server
//    (internal/api/), the same calls instead travel over fetch() to /api/*
//
// isWailsRuntime() lets each exported function choose the correct transport
// at call time, always handing back data in the same shape no matter which
// backend served it — so none of the 18 consuming components need to know
// or care which mode is active. Always import from this module rather than
// reaching into 'wailsjs/go/app/App.js' directly.

import * as Wails from '../../wailsjs/go/app/App.js'

let cachedIsWails: boolean | null = null

// Cached after the first check — a later change to window.go won't flip the result.
export function isWailsRuntime(): boolean {
  if (cachedIsWails === null) {
    const w = window as any
    cachedIsWails = typeof window !== 'undefined' && !!w.go?.app?.App && !!w.runtime
  }
  return cachedIsWails
}

const API_BASE = '/api'

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, init)
  if (!res.ok) {
    let message = `${res.status} ${res.statusText}`
    try {
      const body = await res.json()
      if (body?.error) message = body.error
    } catch {
      // body wasn't JSON, so fall back to the status-based message
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

function qs(params: Record<string, string | number | boolean | undefined>): string {
  const entries = Object.entries(params).filter(([, v]) => v !== undefined && v !== '')
  if (entries.length === 0) return ''
  return '?' + entries.map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`).join('&')
}

// Existing Wails call sites already rely on `any` and optional chaining
// rather than the generated wailsjs classes, so these interfaces are kept
// deliberately loose (with an extra `[key: string]: any`) rather than
// mirroring every backend struct field one-for-one — they only cover the
// fields components actually consume (confirmed by auditing the call sites).

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

export async function AnalyzeMultiple(paths: string[]): Promise<AnalysisResult[]> {
  if (isWailsRuntime()) {
    return Wails.AnalyzeMultiple(paths) as unknown as Promise<AnalysisResult[]>
  }

  const raw = await apiPost<any[]>('/analyze/multiple', { paths })
  // The REST endpoint returns a different shape (isUpscaled/spectralCutoff/
  // message) than core.AnalysisResult (isTrueLossless/spectrumCutoff/
  // details) on purpose — normalize it here so callers see one shape
  // regardless of backend.
  // Known gap: the REST endpoint returns neither filePath nor
  // expectedCutoff. filePath gets rebuilt from the order of the request's
  // paths (the server preserves that order); expectedCutoff is left at 0.
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

export async function CancelDownload(trackId: number): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.CancelDownload(trackId)
  }
  await apiPost(`/downloads/cancel/${trackId}`)
}

// Note: Wails' Pause/ResumeDownloads report whether the call actually
// changed the paused state, whereas the REST endpoints just return a fixed
// paused:true/false no matter what the prior state was. Nothing currently
// reads that return value (call sites update their local store
// optimistically instead), so this is harmless for now — noting it here in
// case a future caller starts depending on it.
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
 * Wails: shows the native "Save As" dialog and hands back the chosen path.
 * Browser: there's no native dialog, so the file is fetched and handed off
 * to the browser's own download mechanism (a throwaway `<a download>` click).
 * Resolves to '' in browser mode — there's no server-side path to report,
 * matching what Wails itself returns when the dialog is cancelled.
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
  // Known gap: unlike Wails, the REST endpoint doesn't yet save a
  // content-level DownloadRecord for contentId/contentType, so History
  // won't show playlist/album progress for downloads queued via the
  // headless server. See migration report.
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

// `records` is deliberately typed `any`: consumers such as History.svelte
// declare their own local DownloadRecord interface, and TypeScript can't
// structurally match two independently-declared interfaces of the same
// name even when an index signature is present.
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

// Deliberately typed `any` here too: MetadataModal.svelte has its own local
// FLACMetadata interface, and the same structural-typing mismatch described
// above for GetDownloadHistoryFiltered applies.
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
  // Known gap: the REST server currently always reports "1.0.0" rather
  // than the actual build version (see migration report).
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

// Known gap: /api/logs and /api/logs/clear on the headless server are
// currently stubs — there's no server-side log buffer wired up yet the way
// the Wails app's logBuffer is — so GetLogs() always resolves to [] in
// browser mode and ClearLogs() does nothing.

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

export async function FetchAndEmbedLyricsMultiple(filePaths: string[]): Promise<any[]> {
  if (isWailsRuntime()) {
    return Wails.FetchAndEmbedLyricsMultiple(filePaths)
  }
  return apiPost('/lyrics/fetch-embed/multiple', { filePaths })
}

// These four Wails calls pop open a native OS dialog (file picker, folder
// picker) or the system file manager. A browser sandbox never has access to
// real filesystem paths — an <input type="file"> only ever hands back a
// filename, never an absolute path — nor to the local file manager, and the
// server might not even share a machine with the browser. There's simply no
// truthful way to implement these in browser mode.
//
// Every call site for these four functions already treats a falsy/empty
// result as "the user cancelled" and no-ops accordingly, so returning that
// same cancelled-style value in browser mode degrades gracefully.

export async function OpenFLACFilesDialog(): Promise<string[]> {
  if (isWailsRuntime()) {
    return Wails.OpenFLACFilesDialog()
  }
  console.warn('OpenFLACFilesDialog: unavailable in browser mode — a browser file picker cannot hand back a server-side file path')
  return []
}

export async function SelectDownloadFolder(): Promise<string> {
  if (isWailsRuntime()) {
    return Wails.SelectDownloadFolder()
  }
  console.warn('SelectDownloadFolder: unavailable in browser mode — browsers have no native folder picker')
  return ''
}

export async function SelectFolderForConversion(): Promise<string[]> {
  if (isWailsRuntime()) {
    return Wails.SelectFolderForConversion()
  }
  console.warn('SelectFolderForConversion: unavailable in browser mode — same limitation as OpenFLACFilesDialog')
  return []
}

export async function SelectFolderForAnalysis(): Promise<string[]> {
  if (isWailsRuntime()) {
    return Wails.SelectFolderForAnalysis()
  }
  console.warn('SelectFolderForAnalysis: unavailable in browser mode — same limitation as OpenFLACFilesDialog')
  return []
}

export async function OpenDownloadFolder(path: string): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.OpenDownloadFolder(path)
  }
  console.warn('OpenDownloadFolder: unavailable in browser mode — a web page cannot reach the local file manager')
}

export async function SetDownloadFolder(folder: string): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.SetDownloadFolder(folder)
  }
  await apiPost('/folder', { folder })
}

/**
 * Known gap: the REST handler, unlike Wails, only saves the config — it
 * doesn't push live settings to the download manager, the downloader's
 * proxy/quality options, or restart the Soulseek source. In browser mode,
 * some settings may not take effect until the server restarts. See the
 * migration report.
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
  console.warn('OpenConfigFolder: unavailable in browser mode — a web page cannot reach the local file manager')
}

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

/**
 * Known gap: /content/fetch on the REST side only handles track/album/
 * playlist through the generic multi-source sourceManager. Wails'
 * FetchTidalContent also handles Tidal "mix" and "artist" URLs, and those
 * will fail here (400/500). See migration report.
 */
export async function FetchTidalContent(url: string): Promise<any> {
  if (isWailsRuntime()) {
    return Wails.FetchTidalContent(url)
  }
  return apiPost('/content/fetch', { url })
}

/**
 * Known gap: the REST check validates against any registered source rather
 * than Tidal specifically, and its success payload uses `contentType`
 * where Wails uses `type` — normalized here to match Wails' shape.
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

/**
 * There's no REST route for this — it's a stateless, side-effect-free call
 * to the public GitHub API, so browser mode calls it directly instead of
 * round-tripping through the server (the same approach About.svelte
 * already uses for repo stats).
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

// Wails mode is unaffected; in browser mode these throw a clear, catchable
// error rather than crashing on an undefined window.go binding or silently
// no-op'ing something the user explicitly triggered (Install buttons,
// discography queueing, etc.). Unlike the native-OS dialogs above, an
// empty/falsy result here could be mistaken for a genuine negative result
// instead of "not implemented", so these throw.

function unavailableInBrowser(name: string, reason: string): never {
  throw new Error(`${name} is unavailable in browser mode: ${reason}`)
}

export async function DownloadArtistAssets(artistId: string, artistName: string, outputDir: string): Promise<number> {
  if (isWailsRuntime()) {
    return Wails.DownloadArtistAssets(artistId, artistName, outputDir)
  }
  return unavailableInBrowser('DownloadArtistAssets', 'a Tidal artist-image client is not yet wired up on the headless server')
}

export async function ExpandDiscographyURL(url: string): Promise<string[]> {
  if (isWailsRuntime()) {
    return Wails.ExpandDiscographyURL(url)
  }
  return unavailableInBrowser('ExpandDiscographyURL', 'a Spotify search client is not yet wired up on the headless server')
}

export async function QueueDiscographyAlbums(albumUrls: string[], outputDir: string): Promise<number> {
  if (isWailsRuntime()) {
    return Wails.QueueDiscographyAlbums(albumUrls, outputDir)
  }
  return unavailableInBrowser('QueueDiscographyAlbums', 'a Spotify search client is not yet wired up on the headless server')
}

export async function CheckAPIStatus(): Promise<any[]> {
  if (isWailsRuntime()) {
    return Wails.CheckAPIStatus()
  }
  return unavailableInBrowser('CheckAPIStatus', 'the headless server does not implement this yet')
}

export async function GetSourceHealth(): Promise<any[]> {
  if (isWailsRuntime()) {
    return Wails.GetSourceHealth()
  }
  return unavailableInBrowser('GetSourceHealth', 'the headless server does not implement this yet (Amazon/Soulseek source wiring is still needed)')
}

export async function InstallFFmpeg(): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.InstallFFmpeg()
  }
  return unavailableInBrowser('InstallFFmpeg', 'installing a binary onto a server that might be remote requires progress-reporting infrastructure that has not been built yet')
}

export async function InstallSldl(): Promise<void> {
  if (isWailsRuntime()) {
    return Wails.InstallSldl()
  }
  return unavailableInBrowser('InstallSldl', 'installing a binary onto a server that might be remote requires progress-reporting infrastructure that has not been built yet')
}
