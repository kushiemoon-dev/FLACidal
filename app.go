package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"flacidal/backend"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct - main Wails application
type App struct {
	ctx             context.Context
	config          *backend.Config
	db              *backend.Database
	tidalClient     *backend.TidalClient
	spotifySearch   *backend.SpotifyClient    // For search/matching (Client Credentials, no login)
	matcher         *backend.Matcher
	downloader      *backend.TidalHifiService // FLAC downloader
	downloadManager *backend.DownloadManager  // Concurrent download manager
	logBuffer       *backend.LogBuffer        // Log buffer for Terminal page
	sourceManager   *backend.SourceManager    // Multi-source manager
	tidalSource     *backend.TidalSource      // Tidal source
	qobuzSource     *backend.QobuzSource      // Qobuz source
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize log buffer
	a.logBuffer = backend.NewLogBuffer(500)
	a.logBuffer.Info("FLACidal starting...")

	// Load config
	config, err := backend.LoadConfig()
	if err != nil {
		println("Warning: Could not load config:", err.Error())
		a.logBuffer.Warn("Could not load config: " + err.Error())
		config = &backend.Config{}
	}
	a.config = config
	a.logBuffer.Success("Configuration loaded")

	// Initialize database
	db, err := backend.NewDatabase()
	if err != nil {
		println("Error: Could not initialize database:", err.Error())
		a.logBuffer.Error("Database initialization failed: " + err.Error())
	} else {
		a.logBuffer.Success("Database initialized")
	}
	a.db = db

	// Initialize Tidal client (uses internal credentials, no user config needed)
	a.tidalClient = backend.NewTidalClientDefault()
	a.logBuffer.Info("Tidal client ready")

	// Initialize Spotify search client (Client Credentials, no login needed)
	a.spotifySearch = backend.NewSpotifyClientForSearch()

	// Initialize matcher
	a.matcher = backend.NewMatcher(a.spotifySearch, a.db)

	// Initialize FLAC downloader
	a.downloader = backend.NewTidalHifiService()
	a.logBuffer.Info("FLAC downloader service ready")

	// Initialize download manager with 4 concurrent workers
	a.downloadManager = backend.NewDownloadManager(a.downloader, 4)
	a.downloadManager.SetProgressCallback(func(trackID int, status string, result *backend.DownloadResult) {
		// Log download events
		if a.logBuffer != nil {
			switch status {
			case "queued":
				a.logBuffer.Info(fmt.Sprintf("Track %d added to queue", trackID))
			case "downloading":
				a.logBuffer.Info(fmt.Sprintf("Downloading track %d...", trackID))
			case "completed":
				if result != nil {
					a.logBuffer.Success(fmt.Sprintf("Downloaded: %s", result.FilePath))
				}
			case "error":
				if result != nil && result.Error != "" {
					a.logBuffer.Error(fmt.Sprintf("Download failed: %s", result.Error))
				}
			case "cancelled":
				a.logBuffer.Warn(fmt.Sprintf("Track %d cancelled", trackID))
			}
		}

		// Emit event to frontend
		runtime.EventsEmit(ctx, "download-progress", map[string]interface{}{
			"trackId": trackID,
			"status":  status,
			"result":  result,
		})
	})
	a.downloadManager.Start()
	a.logBuffer.Success("Download manager started (4 workers)")

	// Initialize source manager
	a.sourceManager = backend.NewSourceManager()

	// Initialize Tidal source
	a.tidalSource = backend.NewTidalSource()
	a.tidalSource.SetAvailable(config.TidalEnabled)
	a.sourceManager.RegisterSource(a.tidalSource)
	a.logBuffer.Info("Tidal source registered")

	// Initialize Qobuz source
	a.qobuzSource = backend.NewQobuzSource(config.QobuzAppID, config.QobuzAppSecret)
	if config.QobuzAuthToken != "" {
		a.qobuzSource.SetCredentials(config.QobuzAppID, config.QobuzAppSecret, config.QobuzAuthToken)
	}
	a.sourceManager.RegisterSource(a.qobuzSource)
	if config.QobuzEnabled && config.QobuzAppID != "" {
		a.logBuffer.Info("Qobuz source registered")
	}

	// Set preferred source
	if config.PreferredSource != "" {
		a.sourceManager.SetPreferredSource(config.PreferredSource)
	}

	a.logBuffer.Success("FLACidal ready!")
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	// Stop download manager
	if a.downloadManager != nil {
		a.downloadManager.Stop()
	}

	// Save config
	if a.config != nil {
		backend.SaveConfig(a.config)
	}

	// Close database
	if a.db != nil {
		a.db.Close()
	}
}

// =============================================================================
// Config Methods (exposed to frontend)
// =============================================================================

// GetConfig returns current configuration
func (a *App) GetConfig() *backend.Config {
	return a.config
}

// SaveConfig saves configuration
func (a *App) SaveConfig(config backend.Config) error {
	a.config = &config
	return backend.SaveConfig(&config)
}

// ResetToDefaults resets configuration to default values
func (a *App) ResetToDefaults() (*backend.Config, error) {
	defaultCfg := backend.GetDefaultConfig()

	// Preserve download folder if set
	if a.config != nil && a.config.DownloadFolder != "" {
		defaultCfg.DownloadFolder = a.config.DownloadFolder
	}

	a.config = defaultCfg
	if err := backend.SaveConfig(defaultCfg); err != nil {
		return nil, err
	}

	if a.logBuffer != nil {
		a.logBuffer.Info("Configuration reset to defaults")
		runtime.EventsEmit(a.ctx, "log", a.logBuffer.Info("Settings restored to defaults"))
	}

	return defaultCfg, nil
}

// GetConnectionStatus returns service status
func (a *App) GetConnectionStatus() map[string]interface{} {
	return map[string]interface{}{
		"tidalReady":    true, // Always ready (uses internal credentials)
		"spotifySearch": a.spotifySearch != nil,
	}
}

// =============================================================================
// Tidal Methods (exposed to frontend)
// =============================================================================

// SetTidalCredentials saves Tidal client credentials
func (a *App) SetTidalCredentials(clientID, clientSecret string) error {
	a.config.TidalClientID = clientID
	a.config.TidalClientSecret = clientSecret

	// Initialize client with new credentials
	a.tidalClient = backend.NewTidalClient(clientID, clientSecret)

	return backend.SaveConfig(a.config)
}

// FetchTidalPlaylist fetches a public playlist from Tidal URL
func (a *App) FetchTidalPlaylist(url string) (*backend.TidalPlaylist, error) {
	// Parse URL to get playlist UUID
	id, contentType, err := backend.ParseTidalURL(url)
	if err != nil {
		return nil, err
	}

	if contentType != "playlist" {
		return nil, fmt.Errorf("URL is not a playlist (got %s)", contentType)
	}

	return a.tidalClient.GetPlaylist(id)
}

// FetchTidalContent fetches playlist, album, or single track from any Tidal URL
func (a *App) FetchTidalContent(url string) (map[string]interface{}, error) {
	id, contentType, err := backend.ParseTidalURL(url)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"type": contentType,
	}

	switch contentType {
	case "playlist":
		playlist, err := a.tidalClient.GetPlaylist(id)
		if err != nil {
			return nil, err
		}
		result["title"] = playlist.Title
		result["creator"] = playlist.Creator
		result["coverUrl"] = playlist.CoverURL
		result["tracks"] = playlist.Tracks
		result["trackCount"] = len(playlist.Tracks)

	case "album":
		album, err := a.tidalClient.GetAlbum(id)
		if err != nil {
			return nil, err
		}
		result["title"] = album.Title
		result["creator"] = album.Artist
		result["coverUrl"] = album.CoverURL
		result["tracks"] = album.Tracks
		result["trackCount"] = len(album.Tracks)

	case "track":
		track, err := a.tidalClient.GetTrack(id)
		if err != nil {
			return nil, err
		}
		result["title"] = track.Title
		result["creator"] = track.Artist
		result["coverUrl"] = track.CoverURL
		result["tracks"] = []backend.TidalTrack{*track}
		result["trackCount"] = 1

	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}

	return result, nil
}

// ValidateTidalURL checks if a URL is a valid Tidal URL
func (a *App) ValidateTidalURL(url string) map[string]interface{} {
	id, contentType, err := backend.ParseTidalURL(url)
	if err != nil {
		return map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		}
	}
	return map[string]interface{}{
		"valid": true,
		"id":    id,
		"type":  contentType,
	}
}

// =============================================================================
// Database Methods (exposed to frontend)
// =============================================================================

// GetCacheStats returns track cache statistics
func (a *App) GetCacheStats() map[string]interface{} {
	if a.db == nil {
		return map[string]interface{}{"error": "database not initialized"}
	}

	total, byMethod, err := a.db.GetCacheStats()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"total":    total,
		"byMethod": byMethod,
	}
}

// GetDownloadHistory returns all download history
func (a *App) GetDownloadHistory() ([]backend.DownloadRecord, error) {
	if a.db == nil {
		return nil, nil
	}
	return a.db.GetAllDownloadRecords()
}

// GetDownloadHistoryFiltered returns filtered download history with pagination
func (a *App) GetDownloadHistoryFiltered(filter map[string]interface{}) (map[string]interface{}, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Parse filter options
	dbFilter := backend.HistoryFilter{}

	if ct, ok := filter["contentType"].(string); ok {
		dbFilter.ContentType = ct
	}
	if search, ok := filter["search"].(string); ok {
		dbFilter.Search = search
	}
	if limit, ok := filter["limit"].(float64); ok {
		dbFilter.Limit = int(limit)
	}
	if offset, ok := filter["offset"].(float64); ok {
		dbFilter.Offset = int(offset)
	}

	records, total, err := a.db.GetDownloadRecordsFiltered(dbFilter)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"records": records,
		"total":   total,
	}, nil
}

// DeleteHistoryRecord deletes a single download history record
func (a *App) DeleteHistoryRecord(id int64) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return a.db.DeleteDownloadRecord(id)
}

// ClearDownloadHistory removes all download history
func (a *App) ClearDownloadHistory() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	err := a.db.ClearAllHistory()
	if err == nil && a.logBuffer != nil {
		a.logBuffer.Info("Download history cleared")
	}
	return err
}

// RefetchFromHistory re-downloads content from history
func (a *App) RefetchFromHistory(tidalContentID string) (map[string]interface{}, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Get the record to find the content type
	record, err := a.db.GetDownloadRecord(tidalContentID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, fmt.Errorf("history record not found")
	}

	// Reconstruct the Tidal URL
	var url string
	switch record.ContentType {
	case "playlist":
		url = fmt.Sprintf("https://tidal.com/browse/playlist/%s", tidalContentID)
	case "album":
		url = fmt.Sprintf("https://tidal.com/browse/album/%s", tidalContentID)
	case "track":
		url = fmt.Sprintf("https://tidal.com/browse/track/%s", tidalContentID)
	default:
		return nil, fmt.Errorf("unknown content type: %s", record.ContentType)
	}

	// Fetch the content
	return a.FetchTidalContent(url)
}

// GetMatchFailures returns all match failures
func (a *App) GetMatchFailures() ([]backend.MatchFailure, error) {
	if a.db == nil {
		return nil, nil
	}
	return a.db.GetMatchFailures()
}

// =============================================================================
// App Info
// =============================================================================

// GetAppVersion returns application version
func (a *App) GetAppVersion() string {
	return "1.0.0"
}

// =============================================================================
// Logging Methods (exposed to frontend)
// =============================================================================

// GetLogs returns all log entries
func (a *App) GetLogs() []backend.LogEntry {
	if a.logBuffer == nil {
		return []backend.LogEntry{}
	}
	return a.logBuffer.GetAll()
}

// ClearLogs clears all log entries
func (a *App) ClearLogs() {
	if a.logBuffer != nil {
		a.logBuffer.Clear()
	}
}

// AddLog adds a log entry (for testing/debug)
func (a *App) AddLog(level, message string) {
	if a.logBuffer != nil {
		entry := a.logBuffer.Add(level, message)
		// Emit log event to frontend
		runtime.EventsEmit(a.ctx, "log", entry)
	}
}

// =============================================================================
// Matcher Methods (exposed to frontend)
// =============================================================================

// MatchPlaylistTracks matches all tracks from a Tidal playlist to Spotify
func (a *App) MatchPlaylistTracks(tracks []backend.TidalTrack) []backend.MatchResult {
	if a.matcher == nil {
		return nil
	}
	return a.matcher.MatchPlaylist(tracks)
}

// MatchSingleTrack matches a single track
func (a *App) MatchSingleTrack(track backend.TidalTrack) backend.MatchResult {
	if a.matcher == nil {
		return backend.MatchResult{TidalTrack: track, Matched: false, MatchMethod: "none"}
	}
	return a.matcher.MatchTrack(track)
}

// =============================================================================
// Download Methods (exposed to frontend)
// =============================================================================

// SelectDownloadFolder opens a folder picker dialog
func (a *App) SelectDownloadFolder() (string, error) {
	folder, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Download Folder",
	})
	if err != nil {
		return "", err
	}
	return folder, nil
}

// GetDownloadFolder returns the configured download folder
func (a *App) GetDownloadFolder() string {
	if a.config != nil && a.config.DownloadFolder != "" {
		return a.config.DownloadFolder
	}
	return ""
}

// SetDownloadFolder saves the download folder to config
func (a *App) SetDownloadFolder(folder string) error {
	if a.config == nil {
		a.config = &backend.Config{}
	}
	a.config.DownloadFolder = folder
	return backend.SaveConfig(a.config)
}

// IsDownloaderAvailable checks if the download service is reachable
func (a *App) IsDownloaderAvailable() bool {
	if a.downloader == nil {
		return false
	}
	return a.downloader.IsAvailable()
}

// DownloadTrack downloads a single track by its Tidal ID
func (a *App) DownloadTrack(trackID int, outputDir string) (*backend.DownloadResult, error) {
	if a.downloader == nil {
		return nil, fmt.Errorf("downloader not initialized")
	}
	if outputDir == "" {
		return nil, fmt.Errorf("no output directory specified")
	}
	return a.downloader.DownloadTrack(trackID, outputDir)
}

// DownloadTrackFromTidal downloads using TidalTrack data (for UI convenience)
func (a *App) DownloadTrackFromTidal(track backend.TidalTrack, outputDir string) (*backend.DownloadResult, error) {
	return a.DownloadTrack(track.ID, outputDir)
}

// QueueDownloads queues multiple tracks for concurrent download
func (a *App) QueueDownloads(tracks []backend.TidalTrack, outputDir string, contentName string) (int, error) {
	if a.downloadManager == nil {
		return 0, fmt.Errorf("download manager not initialized")
	}
	if outputDir == "" {
		return 0, fmt.Errorf("no output directory specified")
	}

	// Create subfolder with content name (playlist/album/track title)
	if contentName != "" {
		outputDir = filepath.Join(outputDir, backend.SanitizeFileName(contentName))
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return 0, fmt.Errorf("failed to create folder: %w", err)
		}
	}

	queued := a.downloadManager.QueueMultiple(tracks, outputDir)
	return queued, nil
}

// QueueSingleDownload queues a single track for download
func (a *App) QueueSingleDownload(trackID int, outputDir, title, artist string) error {
	if a.downloadManager == nil {
		return fmt.Errorf("download manager not initialized")
	}
	if outputDir == "" {
		return fmt.Errorf("no output directory specified")
	}

	return a.downloadManager.QueueDownload(trackID, outputDir, title, artist)
}

// GetDownloadQueueStatus returns current queue status
func (a *App) GetDownloadQueueStatus() map[string]interface{} {
	if a.downloadManager == nil {
		return map[string]interface{}{"running": false}
	}

	return map[string]interface{}{
		"running":     a.downloadManager.IsRunning(),
		"paused":      a.downloadManager.IsPaused(),
		"activeCount": a.downloadManager.GetActiveCount(),
		"queueLength": a.downloadManager.GetQueueLength(),
	}
}

// GetDownloadOptions returns current download options
func (a *App) GetDownloadOptions() map[string]interface{} {
	if a.config == nil {
		return map[string]interface{}{
			"quality":         "LOSSLESS",
			"fileNameFormat":  "{artist} - {title}",
			"organizeFolders": false,
			"embedCover":      true,
		}
	}

	quality := a.config.DownloadQuality
	if quality == "" {
		quality = "LOSSLESS"
	}
	format := a.config.FileNameFormat
	if format == "" {
		format = "{artist} - {title}"
	}

	return map[string]interface{}{
		"quality":         quality,
		"fileNameFormat":  format,
		"organizeFolders": a.config.OrganizeFolders,
		"embedCover":      a.config.EmbedCover,
	}
}

// SetDownloadOptions updates download options
func (a *App) SetDownloadOptions(quality, fileNameFormat string, organizeFolders, embedCover bool) error {
	if a.config == nil {
		a.config = &backend.Config{}
	}

	a.config.DownloadQuality = quality
	a.config.FileNameFormat = fileNameFormat
	a.config.OrganizeFolders = organizeFolders
	a.config.EmbedCover = embedCover

	// Update downloader options
	if a.downloader != nil {
		a.downloader.SetOptions(backend.DownloadOptions{
			Quality:         quality,
			FileNameFormat:  fileNameFormat,
			OrganizeFolders: organizeFolders,
			EmbedCover:      embedCover,
		})
	}

	return backend.SaveConfig(a.config)
}

// OpenDownloadFolder opens the download folder in the system file manager
func (a *App) OpenDownloadFolder(folder string) error {
	if folder == "" {
		return fmt.Errorf("no folder specified")
	}
	runtime.BrowserOpenURL(a.ctx, "file://"+folder)
	return nil
}

// =============================================================================
// Search Methods (exposed to frontend)
// =============================================================================

// SearchTidal searches for tracks on Tidal
func (a *App) SearchTidal(query string) ([]backend.TidalTrack, error) {
	if a.downloader == nil {
		return nil, fmt.Errorf("downloader not initialized")
	}

	results, err := a.downloader.SearchTracks(query, 50)
	if err != nil {
		return nil, err
	}

	// Convert to TidalTrack format for frontend
	tracks := make([]backend.TidalTrack, len(results))
	for i, r := range results {
		// Build artist string
		var artists []string
		for _, art := range r.Artists {
			artists = append(artists, art.Name)
		}
		artistStr := ""
		if len(artists) > 0 {
			artistStr = artists[0]
		}
		allArtists := artistStr
		if len(artists) > 1 {
			allArtists = fmt.Sprintf("%s, %s", artists[0], artists[1])
			if len(artists) > 2 {
				allArtists += fmt.Sprintf(" +%d", len(artists)-2)
			}
		}

		// Build cover URL
		coverURL := ""
		if r.Album.Cover != "" {
			coverURL = fmt.Sprintf("https://resources.tidal.com/images/%s/320x320.jpg",
				backend.FormatCoverUUID(r.Album.Cover))
		}

		tracks[i] = backend.TidalTrack{
			ID:       r.ID,
			Title:    r.Title,
			Artist:   artistStr,
			Artists:  allArtists,
			Album:    r.Album.Title,
			Duration: r.Duration,
			ISRC:     r.ISRC,
			CoverURL: coverURL,
			Explicit: r.Explicit,
		}
	}

	return tracks, nil
}

// =============================================================================
// File Browser Methods (exposed to frontend)
// =============================================================================

// ListDownloadedFiles lists all downloaded FLAC files
func (a *App) ListDownloadedFiles() ([]backend.DownloadedFileInfo, error) {
	folder := a.GetDownloadFolder()
	if folder == "" {
		return []backend.DownloadedFileInfo{}, nil
	}

	return backend.ListFLACFiles(folder)
}

// DeleteFile deletes a file from the filesystem
func (a *App) DeleteFile(path string) error {
	return backend.DeleteFile(path)
}

// GetFileMetadata reads and returns metadata from a FLAC file
func (a *App) GetFileMetadata(filePath string) (*backend.FLACMetadata, error) {
	return backend.ReadFLACMetadata(filePath)
}

// GetFileCoverArt returns cover art as base64 encoded string
func (a *App) GetFileCoverArt(filePath string) (map[string]string, error) {
	base64Data, mimeType, err := backend.GetCoverArtBase64(filePath)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"data":     base64Data,
		"mimeType": mimeType,
	}, nil
}

// GetRenameTemplates returns available rename templates
func (a *App) GetRenameTemplates() []map[string]string {
	return backend.GetRenameTemplates()
}

// PreviewRename generates a preview of rename operations
func (a *App) PreviewRename(files []string, template string) []backend.RenamePreview {
	return backend.PreviewRename(files, template)
}

// RenameFiles renames files according to the template
func (a *App) RenameFiles(files []string, template string) []backend.RenameResult {
	results := backend.RenameFiles(files, template)

	// Log results
	if a.logBuffer != nil {
		success := 0
		failed := 0
		for _, r := range results {
			if r.Success {
				success++
			} else {
				failed++
			}
		}
		a.logBuffer.Info(fmt.Sprintf("Renamed %d files (%d failed)", success, failed))
	}

	return results
}

// =============================================================================
// Converter Methods (exposed to frontend)
// =============================================================================

// IsConverterAvailable checks if FFmpeg is available
func (a *App) IsConverterAvailable() bool {
	return backend.IsConverterAvailable()
}

// GetFFmpegInfo returns FFmpeg availability and version
func (a *App) GetFFmpegInfo() map[string]interface{} {
	return backend.GetFFmpegInfo()
}

// GetConversionFormats returns available conversion formats
func (a *App) GetConversionFormats() []backend.ConversionFormat {
	conv := backend.GetConverter()
	if conv == nil {
		return []backend.ConversionFormat{}
	}
	return conv.GetFormats()
}

// ConvertFiles converts files to the specified format
func (a *App) ConvertFiles(files []string, format, quality, outputDir string, deleteSource bool) []backend.ConversionResult {
	conv := backend.GetConverter()
	if conv == nil {
		results := make([]backend.ConversionResult, len(files))
		for i, f := range files {
			results[i] = backend.ConversionResult{
				SourcePath: f,
				Error:      "FFmpeg not available",
			}
		}
		return results
	}

	opts := backend.ConversionOptions{
		Format:       format,
		Quality:      quality,
		OutputDir:    outputDir,
		DeleteSource: deleteSource,
	}

	results := conv.ConvertMultiple(files, opts)

	// Log results
	if a.logBuffer != nil {
		success := 0
		for _, r := range results {
			if r.Success {
				success++
			}
		}
		a.logBuffer.Info(fmt.Sprintf("Converted %d/%d files to %s", success, len(files), format))
	}

	return results
}

// OpenInFileManager opens the file's directory in the system file manager
func (a *App) OpenInFileManager(path string) error {
	runtime.BrowserOpenURL(a.ctx, "file://"+path)
	return nil
}

// RetryDownload retries a failed download
func (a *App) RetryDownload(trackID int) error {
	if a.downloadManager == nil {
		return fmt.Errorf("download manager not initialized")
	}

	folder := a.GetDownloadFolder()
	if folder == "" {
		return fmt.Errorf("no download folder configured")
	}

	return a.downloadManager.QueueDownload(trackID, folder, "", "")
}

// RetryAllFailed retries all failed downloads
func (a *App) RetryAllFailed() (int, error) {
	if a.downloadManager == nil {
		return 0, fmt.Errorf("download manager not initialized")
	}

	count := a.downloadManager.RetryAllFailed()
	return count, nil
}

// CancelDownload cancels a download in progress
func (a *App) CancelDownload(trackID int) error {
	if a.downloadManager == nil {
		return fmt.Errorf("download manager not initialized")
	}

	return a.downloadManager.CancelDownload(trackID)
}

// PauseDownloads pauses the download queue
func (a *App) PauseDownloads() bool {
	if a.downloadManager == nil {
		return false
	}

	success := a.downloadManager.PauseQueue()
	if success && a.logBuffer != nil {
		a.logBuffer.Info("Download queue paused")
		runtime.EventsEmit(a.ctx, "queue-paused", true)
	}
	return success
}

// ResumeDownloads resumes the download queue
func (a *App) ResumeDownloads() bool {
	if a.downloadManager == nil {
		return false
	}

	success := a.downloadManager.ResumeQueue()
	if success && a.logBuffer != nil {
		a.logBuffer.Info("Download queue resumed")
		runtime.EventsEmit(a.ctx, "queue-paused", false)
	}
	return success
}

// IsQueuePaused returns whether the download queue is paused
func (a *App) IsQueuePaused() bool {
	if a.downloadManager == nil {
		return false
	}
	return a.downloadManager.IsPaused()
}

// =============================================================================
// Analyzer Methods (exposed to frontend)
// =============================================================================

// AnalyzeFile analyzes a single FLAC file for quality/authenticity
func (a *App) AnalyzeFile(filePath string) (*backend.AnalysisResult, error) {
	result, err := backend.AnalyzeFLAC(filePath)
	if err != nil {
		return nil, err
	}

	if a.logBuffer != nil {
		a.logBuffer.Info(fmt.Sprintf("Analyzed: %s - %s", result.FileName, result.VerdictLabel))
	}

	return result, nil
}

// AnalyzeMultiple analyzes multiple files
func (a *App) AnalyzeMultiple(filePaths []string) []backend.AnalysisResult {
	results := backend.AnalyzeMultiple(filePaths)

	if a.logBuffer != nil {
		lossless := 0
		upscaled := 0
		for _, r := range results {
			if r.IsTrueLossless {
				lossless++
			} else if r.Verdict != "error" {
				upscaled++
			}
		}
		a.logBuffer.Info(fmt.Sprintf("Analyzed %d files: %d lossless, %d upscaled", len(results), lossless, upscaled))
	}

	return results
}

// QuickAnalyze performs a fast analysis based on file size heuristics
func (a *App) QuickAnalyze(filePath string) (*backend.AnalysisResult, error) {
	return backend.QuickAnalyze(filePath)
}

// =============================================================================
// Lyrics Methods (exposed to frontend)
// =============================================================================

// FetchLyrics fetches lyrics for a track from LRCLIB
func (a *App) FetchLyrics(title, artist string, durationSec int) (*backend.Lyrics, error) {
	client := backend.NewLyricsClient()
	lyrics, err := client.SearchLyrics(title, artist, durationSec)
	if err != nil {
		if a.logBuffer != nil {
			a.logBuffer.Warn(fmt.Sprintf("Lyrics not found for %s - %s", artist, title))
		}
		return nil, err
	}

	if a.logBuffer != nil {
		if lyrics.HasSynced {
			a.logBuffer.Success(fmt.Sprintf("Found synced lyrics for %s - %s", artist, title))
		} else {
			a.logBuffer.Success(fmt.Sprintf("Found plain lyrics for %s - %s", artist, title))
		}
	}

	return lyrics, nil
}

// FetchLyricsForFile fetches lyrics based on a FLAC file's metadata
func (a *App) FetchLyricsForFile(filePath string) (*backend.Lyrics, error) {
	meta, err := backend.ReadFLACMetadata(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	client := backend.NewLyricsClient()
	return client.FetchLyricsForFile(meta)
}

// EmbedLyricsToFile embeds lyrics into a FLAC file
func (a *App) EmbedLyricsToFile(filePath string, plain, synced string) error {
	tagger := backend.NewFLACTagger()
	err := tagger.EmbedLyrics(filePath, plain, synced)
	if err != nil {
		if a.logBuffer != nil {
			a.logBuffer.Error(fmt.Sprintf("Failed to embed lyrics: %s", err.Error()))
		}
		return err
	}

	if a.logBuffer != nil {
		a.logBuffer.Success(fmt.Sprintf("Lyrics embedded to %s", filepath.Base(filePath)))
	}
	return nil
}

// FetchAndEmbedLyrics fetches and embeds lyrics for a file in one operation
func (a *App) FetchAndEmbedLyrics(filePath string) (*backend.Lyrics, error) {
	// Fetch lyrics based on file metadata
	lyrics, err := a.FetchLyricsForFile(filePath)
	if err != nil {
		return nil, err
	}

	// Embed lyrics
	err = a.EmbedLyricsToFile(filePath, lyrics.Plain, lyrics.Synced)
	if err != nil {
		return lyrics, err // Return lyrics even if embedding failed
	}

	return lyrics, nil
}

// FetchAndEmbedLyricsMultiple fetches and embeds lyrics for multiple files
func (a *App) FetchAndEmbedLyricsMultiple(filePaths []string) []map[string]interface{} {
	results := make([]map[string]interface{}, len(filePaths))

	for i, filePath := range filePaths {
		result := map[string]interface{}{
			"filePath": filePath,
			"success":  false,
		}

		lyrics, err := a.FetchAndEmbedLyrics(filePath)
		if err != nil {
			result["error"] = err.Error()
		} else {
			result["success"] = true
			result["hasPlain"] = lyrics.Plain != ""
			result["hasSynced"] = lyrics.HasSynced
		}

		results[i] = result
	}

	return results
}

// =============================================================================
// Source Manager Methods (exposed to frontend)
// =============================================================================

// GetAvailableSources returns info about all registered music sources
func (a *App) GetAvailableSources() []backend.SourceInfo {
	return a.sourceManager.GetSourcesInfo()
}

// GetPreferredSource returns the currently preferred source name
func (a *App) GetPreferredSource() string {
	source, ok := a.sourceManager.GetPreferredSource()
	if ok {
		return source.Name()
	}
	return "tidal"
}

// SetPreferredSource sets the preferred source
func (a *App) SetPreferredSource(sourceName string) {
	a.sourceManager.SetPreferredSource(sourceName)
	if a.logBuffer != nil {
		a.logBuffer.Info(fmt.Sprintf("Preferred source set to: %s", sourceName))
	}
}

// DetectSourceFromURL identifies which source can handle a URL
func (a *App) DetectSourceFromURL(rawURL string) map[string]interface{} {
	result := map[string]interface{}{
		"detected":    false,
		"source":      "",
		"displayName": "",
		"contentType": "",
		"id":          "",
		"available":   false,
	}

	source, err := a.sourceManager.DetectSource(rawURL)
	if err != nil {
		return result
	}

	id, contentType, err := source.ParseURL(rawURL)
	if err != nil {
		return result
	}

	result["detected"] = true
	result["source"] = source.Name()
	result["displayName"] = source.DisplayName()
	result["contentType"] = contentType
	result["id"] = id
	result["available"] = source.IsAvailable()

	return result
}

// FetchContentFromURL fetches content info from any supported source URL
func (a *App) FetchContentFromURL(rawURL string) (map[string]interface{}, error) {
	source, err := a.sourceManager.DetectSource(rawURL)
	if err != nil {
		return nil, err
	}

	id, contentType, err := source.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"source": source.Name(),
		"type":   contentType,
		"id":     id,
	}

	// Helper to convert SourceTrack to frontend-compatible format
	convertTrack := func(t backend.SourceTrack) map[string]interface{} {
		// Convert ID to int if possible, otherwise use string
		trackID, _ := strconv.Atoi(t.ID)
		artists := t.Artist
		if len(t.Artists) > 0 {
			artists = strings.Join(t.Artists, ", ")
		}
		return map[string]interface{}{
			"id":          trackID,
			"title":       t.Title,
			"artist":      t.Artist,
			"artists":     artists,
			"album":       t.Album,
			"duration":    t.Duration,
			"trackNumber": t.TrackNumber,
			"coverUrl":    t.CoverURL,
			"explicit":    t.Explicit,
			"isrc":        t.ISRC,
		}
	}

	convertTracks := func(tracks []backend.SourceTrack) []map[string]interface{} {
		result := make([]map[string]interface{}, len(tracks))
		for i, t := range tracks {
			result[i] = convertTrack(t)
		}
		return result
	}

	switch contentType {
	case "track":
		track, err := source.GetTrack(id)
		if err != nil {
			return nil, err
		}
		result["title"] = track.Title
		result["creator"] = track.Artist
		result["coverUrl"] = track.CoverURL
		result["tracks"] = convertTracks([]backend.SourceTrack{*track})

	case "album":
		album, err := source.GetAlbum(id)
		if err != nil {
			return nil, err
		}
		result["title"] = album.Title
		result["creator"] = album.Artist
		result["coverUrl"] = album.CoverURL
		result["tracks"] = convertTracks(album.Tracks)

	case "playlist":
		playlist, err := source.GetPlaylist(id)
		if err != nil {
			return nil, err
		}
		result["title"] = playlist.Title
		result["creator"] = playlist.Creator
		result["coverUrl"] = playlist.CoverURL
		result["tracks"] = convertTracks(playlist.Tracks)
	}

	if a.logBuffer != nil {
		a.logBuffer.Info(fmt.Sprintf("Fetched %s from %s: %s", contentType, source.DisplayName(), id))
	}

	return result, nil
}

// GetSourceTrack fetches a track from a specific source
func (a *App) GetSourceTrack(sourceName, trackID string) (*backend.SourceTrack, error) {
	source, ok := a.sourceManager.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("source not found: %s", sourceName)
	}
	return source.GetTrack(trackID)
}

// GetSourceAlbum fetches an album from a specific source
func (a *App) GetSourceAlbum(sourceName, albumID string) (*backend.SourceAlbum, error) {
	source, ok := a.sourceManager.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("source not found: %s", sourceName)
	}
	return source.GetAlbum(albumID)
}

// GetSourcePlaylist fetches a playlist from a specific source
func (a *App) GetSourcePlaylist(sourceName, playlistID string) (*backend.SourcePlaylist, error) {
	source, ok := a.sourceManager.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("source not found: %s", sourceName)
	}
	return source.GetPlaylist(playlistID)
}

// UpdateQobuzCredentials updates Qobuz credentials
func (a *App) UpdateQobuzCredentials(appID, appSecret, authToken string) error {
	a.qobuzSource.SetCredentials(appID, appSecret, authToken)

	// Update config
	a.config.QobuzAppID = appID
	a.config.QobuzAppSecret = appSecret
	a.config.QobuzAuthToken = authToken
	a.config.QobuzEnabled = appID != "" && appSecret != ""

	if err := backend.SaveConfig(a.config); err != nil {
		return err
	}

	if a.logBuffer != nil {
		if a.config.QobuzEnabled {
			a.logBuffer.Success("Qobuz credentials updated")
		} else {
			a.logBuffer.Info("Qobuz disabled")
		}
	}

	return nil
}

// IsQobuzConfigured checks if Qobuz is properly configured
func (a *App) IsQobuzConfigured() bool {
	return a.qobuzSource.IsAvailable()
}
