package api

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"

	"flacidal/internal/app"
)

// Health check
func (s *Server) handleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"service": "flacidal",
	})
}

// Config handlers
func (s *Server) handleGetConfig(c *fiber.Ctx) error {
	return c.JSON(s.config)
}

func (s *Server) handleSaveConfig(c *fiber.Ctx) error {
	var config core.Config
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if err := core.SaveConfig(&config); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	s.config = &config
	return c.JSON(fiber.Map{"success": true})
}

func (s *Server) handleResetConfig(c *fiber.Ctx) error {
	config := core.GetDefaultConfig()

	// Preserve download folder if set — mirrors internal/app's App.ResetToDefaults.
	if s.config != nil && s.config.DownloadFolder != "" {
		config.DownloadFolder = s.config.DownloadFolder
	}

	if err := core.SaveConfig(config); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	s.config = config
	return c.JSON(config)
}

// Source handlers
func (s *Server) handleGetSources(c *fiber.Ctx) error {
	return c.JSON(s.sourceManager.GetSourcesInfo())
}

func (s *Server) handleGetPreferredSource(c *fiber.Ctx) error {
	source, ok := s.sourceManager.GetPreferredSource()
	if !ok {
		return c.JSON(fiber.Map{"source": ""})
	}
	return c.JSON(fiber.Map{"source": source.Name()})
}

func (s *Server) handleSetPreferredSource(c *fiber.Ctx) error {
	var req struct {
		Source string `json:"source"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	s.sourceManager.SetPreferredSource(req.Source)
	return c.JSON(fiber.Map{"success": true})
}

func (s *Server) handleDetectSource(c *fiber.Ctx) error {
	var req struct {
		URL string `json:"url"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	source, err := s.sourceManager.DetectSource(req.URL)
	if err != nil {
		return c.JSON(fiber.Map{
			"detected":    false,
			"source":      "",
			"displayName": "",
			"available":   false,
		})
	}

	id, contentType, _ := source.ParseURL(req.URL)
	return c.JSON(fiber.Map{
		"detected":    true,
		"source":      source.Name(),
		"displayName": source.DisplayName(),
		"contentType": contentType,
		"id":          id,
		"available":   source.IsAvailable(),
	})
}

// Content handlers
func (s *Server) handleFetchContent(c *fiber.Ctx) error {
	var req struct {
		URL string `json:"url"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	result, status, err := s.fetchContentByURL(req.URL)
	if err != nil {
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// fetchContentByURL resolves a source URL (Tidal/Qobuz track, album or
// playlist) into its details. Shared by handleFetchContent and
// handleRefetchFromHistory so both stay in sync.
func (s *Server) fetchContentByURL(rawURL string) (fiber.Map, int, error) {
	source, err := s.sourceManager.DetectSource(rawURL)
	if err != nil {
		return nil, fiber.StatusBadRequest, fmt.Errorf("Unknown URL format")
	}

	id, contentType, err := source.ParseURL(rawURL)
	if err != nil {
		return nil, fiber.StatusBadRequest, err
	}

	result := fiber.Map{
		"source": source.Name(),
		"type":   contentType,
		"id":     id,
	}

	switch contentType {
	case "track":
		track, err := source.GetTrack(id)
		if err != nil {
			return nil, fiber.StatusInternalServerError, err
		}
		result["title"] = track.Title
		result["creator"] = track.Artist
		result["coverUrl"] = track.CoverURL
		result["tracks"] = []core.SourceTrack{*track}

	case "album":
		album, err := source.GetAlbum(id)
		if err != nil {
			return nil, fiber.StatusInternalServerError, err
		}
		result["title"] = album.Title
		result["creator"] = album.Artist
		result["coverUrl"] = album.CoverURL
		result["tracks"] = album.Tracks

	case "playlist":
		playlist, err := source.GetPlaylist(id)
		if err != nil {
			return nil, fiber.StatusInternalServerError, err
		}
		result["title"] = playlist.Title
		result["creator"] = playlist.Creator
		result["coverUrl"] = playlist.CoverURL
		result["tracks"] = playlist.Tracks
	}

	return result, fiber.StatusOK, nil
}

func (s *Server) handleValidateURL(c *fiber.Ctx) error {
	var req struct {
		URL string `json:"url"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	source, err := s.sourceManager.DetectSource(req.URL)
	if err != nil {
		return c.JSON(fiber.Map{"valid": false, "error": "Unknown URL format"})
	}

	id, contentType, err := source.ParseURL(req.URL)
	if err != nil {
		return c.JSON(fiber.Map{"valid": false, "error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"valid":       true,
		"source":      source.Name(),
		"contentType": contentType,
		"id":          id,
	})
}

func (s *Server) handleSearch(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Query parameter 'q' is required"})
	}
	if s.tidalSource == nil || s.tidalSource.GetService() == nil {
		return c.Status(500).JSON(fiber.Map{"error": "downloader not initialized"})
	}

	limit, err := strconv.Atoi(c.Query("limit", "50"))
	if err != nil || limit <= 0 {
		limit = 50
	}

	results, err := s.tidalSource.GetService().SearchTracks(query, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(app.ConvertTidalSearchResults(results))
}

// Download handlers
func (s *Server) handleGetQueue(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"active":  s.downloadManager.GetActiveCount(),
		"queued":  s.downloadManager.GetQueueLength(),
		"failed":  s.downloadManager.GetFailedCount(),
		"running": s.downloadManager.IsRunning(),
		"paused":  s.downloadManager.IsPaused(),
	})
}

func (s *Server) handleQueueDownloads(c *fiber.Ctx) error {
	var req struct {
		Tracks      []core.TidalTrack `json:"tracks"`
		OutputDir   string            `json:"outputDir"`
		ContentName string            `json:"contentName"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	outputDir := req.OutputDir
	if outputDir == "" {
		outputDir = s.config.DownloadFolder
	}
	if outputDir == "" {
		outputDir = core.GetDefaultDownloadFolder()
	}

	count := s.downloadManager.QueueMultiple(req.Tracks, outputDir)
	return c.JSON(fiber.Map{"queued": count})
}

func (s *Server) handleQueueSingle(c *fiber.Ctx) error {
	var req struct {
		TrackID   int    `json:"trackId"`
		OutputDir string `json:"outputDir"`
		Title     string `json:"title"`
		Artist    string `json:"artist"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	outputDir := req.OutputDir
	if outputDir == "" {
		outputDir = s.config.DownloadFolder
	}
	if outputDir == "" {
		outputDir = core.GetDefaultDownloadFolder()
	}

	err := s.downloadManager.QueueDownload(req.TrackID, outputDir, req.Title, req.Artist)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

func (s *Server) handleGetQueueStatus(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"running":     s.downloadManager.IsRunning(),
		"paused":      s.downloadManager.IsPaused(),
		"activeCount": s.downloadManager.GetActiveCount(),
		"queueLength": s.downloadManager.GetQueueLength(),
		"failedCount": s.downloadManager.GetFailedCount(),
	})
}

func (s *Server) handleGetDownloadOptions(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"quality":         s.config.DownloadQuality,
		"fileNameFormat":  s.config.FileNameFormat,
		"organizeFolders": s.config.OrganizeFolders,
		"embedCover":      s.config.EmbedCover,
		"saveCoverFile":   s.config.SaveCoverFile,
		"saveFolderCover": s.config.SaveFolderCover,
		"autoAnalyze":     s.config.AutoAnalyze,
		"embedLyrics":     s.config.EmbedLyrics,
	})
}

func (s *Server) handleSetDownloadOptions(c *fiber.Ctx) error {
	var req struct {
		Quality         string `json:"quality"`
		FileNameFormat  string `json:"fileNameFormat"`
		OrganizeFolders bool   `json:"organizeFolders"`
		EmbedCover      bool   `json:"embedCover"`
		SaveCoverFile   bool   `json:"saveCoverFile"`
		AutoAnalyze     bool   `json:"autoAnalyze"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	s.config.DownloadQuality = req.Quality
	s.config.FileNameFormat = req.FileNameFormat
	s.config.OrganizeFolders = req.OrganizeFolders
	s.config.EmbedCover = req.EmbedCover
	s.config.SaveCoverFile = req.SaveCoverFile
	s.config.AutoAnalyze = req.AutoAnalyze

	if err := core.SaveConfig(s.config); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

func (s *Server) handleRetryDownload(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid ID"})
	}

	// Re-queue the download - the download manager tracks failed jobs internally
	outputDir := s.config.DownloadFolder
	if outputDir == "" {
		outputDir = core.GetDefaultDownloadFolder()
	}
	if err := s.downloadManager.QueueDownload(id, outputDir, "", ""); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

func (s *Server) handleRetryAllFailed(c *fiber.Ctx) error {
	count := s.downloadManager.RetryAllFailed()
	return c.JSON(fiber.Map{"retried": count})
}

func (s *Server) handleCancelDownload(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid ID"})
	}

	if err := s.downloadManager.CancelDownload(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

func (s *Server) handlePauseDownloads(c *fiber.Ctx) error {
	s.downloadManager.PauseQueue()
	return c.JSON(fiber.Map{"paused": true})
}

func (s *Server) handleResumeDownloads(c *fiber.Ctx) error {
	s.downloadManager.ResumeQueue()
	return c.JSON(fiber.Map{"paused": false})
}

func (s *Server) handleIsPaused(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"paused": s.downloadManager.IsPaused()})
}

// History handlers
func (s *Server) handleGetHistory(c *fiber.Ctx) error {
	if s.db == nil {
		return c.JSON([]core.DownloadRecord{})
	}

	records, err := s.db.GetAllDownloadRecords()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(records)
}

func (s *Server) handleGetHistoryFiltered(c *fiber.Ctx) error {
	// Parse filter from query params
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	if s.db == nil {
		return c.JSON(fiber.Map{"records": []core.DownloadRecord{}, "total": 0})
	}

	filter := core.HistoryFilter{
		ContentType: c.Query("contentType"),
		Search:      c.Query("search"),
		Limit:       limit,
		Offset:      offset,
	}
	records, total, err := s.db.GetDownloadRecordsFiltered(filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"records": records,
		"total":   total,
	})
}

func (s *Server) handleDeleteHistory(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid ID"})
	}

	if s.db == nil {
		return c.Status(500).JSON(fiber.Map{"error": "Database not available"})
	}

	if err := s.db.DeleteDownloadRecord(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

func (s *Server) handleClearHistory(c *fiber.Ctx) error {
	if s.db == nil {
		return c.Status(500).JSON(fiber.Map{"error": "Database not available"})
	}

	if err := s.db.ClearAllHistory(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

// handleRefetchFromHistory re-resolves a history record's content from its
// source URL. Mirrors internal/app's App.RefetchFromHistory.
func (s *Server) handleRefetchFromHistory(c *fiber.Ctx) error {
	tidalContentID := c.Params("id")

	if s.db == nil {
		return c.Status(500).JSON(fiber.Map{"error": "Database not available"})
	}

	record, err := s.db.GetDownloadRecord(tidalContentID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if record == nil {
		return c.Status(404).JSON(fiber.Map{"error": "history record not found"})
	}

	var url string
	switch record.ContentType {
	case "playlist":
		url = fmt.Sprintf("https://tidal.com/browse/playlist/%s", tidalContentID)
	case "album":
		url = fmt.Sprintf("https://tidal.com/browse/album/%s", tidalContentID)
	case "track":
		url = fmt.Sprintf("https://tidal.com/browse/track/%s", tidalContentID)
	default:
		return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("unknown content type: %s", record.ContentType)})
	}

	result, status, err := s.fetchContentByURL(url)
	if err != nil {
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// Files handlers
func (s *Server) handleListFiles(c *fiber.Ctx) error {
	folder := s.config.DownloadFolder
	if folder == "" {
		return c.JSON([]core.DownloadedFileInfo{})
	}

	files, err := core.ListFLACFiles(folder)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(files)
}

func (s *Server) handleDeleteFile(c *fiber.Ctx) error {
	path := c.Query("path")
	if path == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Path required"})
	}

	if err := os.Remove(path); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

func (s *Server) handleGetMetadata(c *fiber.Ctx) error {
	path := c.Query("path")
	if path == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Path required"})
	}

	meta, err := core.ReadFLACMetadata(path)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(meta)
}

func (s *Server) handleGetCoverArt(c *fiber.Ctx) error {
	path := c.Query("path")
	if path == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Path required"})
	}

	base64Data, mimeType, err := core.GetCoverArtBase64(path)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"data": base64Data, "mimeType": mimeType})
}

func (s *Server) handleGetRenameTemplates(c *fiber.Ctx) error {
	return c.JSON(core.GetRenameTemplates())
}

func (s *Server) handlePreviewRename(c *fiber.Ctx) error {
	var req struct {
		Files    []string `json:"files"`
		Template string   `json:"template"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	previews := core.PreviewRename(req.Files, req.Template)
	return c.JSON(previews)
}

func (s *Server) handleRenameFiles(c *fiber.Ctx) error {
	var req struct {
		Files    []string `json:"files"`
		Template string   `json:"template"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	results := core.RenameFiles(req.Files, req.Template)
	return c.JSON(results)
}

// Conversion handlers
func (s *Server) handleIsConverterAvailable(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"available": core.IsConverterAvailable()})
}

func (s *Server) handleGetFFmpegInfo(c *fiber.Ctx) error {
	return c.JSON(core.GetFFmpegInfo())
}

func (s *Server) handleGetConversionFormats(c *fiber.Ctx) error {
	conv := core.GetConverter()
	if conv == nil {
		return c.JSON([]core.ConversionFormat{})
	}
	return c.JSON(conv.GetFormats())
}

func (s *Server) handleConvertFiles(c *fiber.Ctx) error {
	var req struct {
		Files        []string `json:"files"`
		Format       string   `json:"format"`
		Quality      string   `json:"quality"`
		OutputDir    string   `json:"outputDir"`
		DeleteSource bool     `json:"deleteSource"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	conv := core.GetConverter()
	if conv == nil {
		results := make([]core.ConversionResult, len(req.Files))
		for i, f := range req.Files {
			results[i] = core.ConversionResult{
				SourcePath: f,
				Error:      "FFmpeg not available",
			}
		}
		return c.JSON(results)
	}

	opts := core.ConversionOptions{
		Format:       req.Format,
		Quality:      req.Quality,
		OutputDir:    req.OutputDir,
		DeleteSource: req.DeleteSource,
	}

	return c.JSON(conv.ConvertMultiple(req.Files, opts))
}

// Lyrics handlers
func (s *Server) handleFetchLyrics(c *fiber.Ctx) error {
	title := c.Query("title")
	artist := c.Query("artist")
	duration, _ := strconv.Atoi(c.Query("duration", "0"))

	if title == "" || artist == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Title and artist required"})
	}

	lyrics, err := s.lyricsClient.SearchLyrics(title, artist, duration)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(lyrics)
}

func (s *Server) handleFetchLyricsForFile(c *fiber.Ctx) error {
	var req struct {
		FilePath string `json:"filePath"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if req.FilePath == "" {
		return c.Status(400).JSON(fiber.Map{"error": "File path required"})
	}

	lyrics, err := s.fetchLyricsForFile(req.FilePath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(lyrics)
}

// fetchLyricsForFile reads a FLAC file's metadata and looks up matching
// lyrics. Mirrors internal/app's App.FetchLyricsForFile.
func (s *Server) fetchLyricsForFile(filePath string) (*core.Lyrics, error) {
	meta, err := core.ReadFLACMetadata(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}
	return s.lyricsClient.FetchLyricsForFile(meta)
}

func (s *Server) handleEmbedLyrics(c *fiber.Ctx) error {
	var req struct {
		FilePath string `json:"filePath"`
		Plain    string `json:"plain"`
		Synced   string `json:"synced"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	tagger := core.NewFLACTagger()
	if err := tagger.EmbedLyrics(req.FilePath, req.Plain, req.Synced); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

func (s *Server) handleFetchAndEmbedLyrics(c *fiber.Ctx) error {
	var req struct {
		FilePath string `json:"filePath"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if req.FilePath == "" {
		return c.Status(400).JSON(fiber.Map{"error": "File path required"})
	}

	lyrics, err := s.fetchAndEmbedLyrics(req.FilePath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(lyrics)
}

// fetchAndEmbedLyrics fetches lyrics for a file and embeds them, optionally
// saving a sidecar .lrc/.txt file when enabled in config. Mirrors
// internal/app's App.FetchAndEmbedLyrics.
func (s *Server) fetchAndEmbedLyrics(filePath string) (*core.Lyrics, error) {
	lyrics, err := s.fetchLyricsForFile(filePath)
	if err != nil {
		return nil, err
	}

	tagger := core.NewFLACTagger()
	if err := tagger.EmbedLyrics(filePath, lyrics.Plain, lyrics.Synced); err != nil {
		return lyrics, err
	}

	if s.config != nil && s.config.SaveLyricsFile {
		core.SaveLyricsFile(filePath, lyrics.Synced, lyrics.Plain) //nolint:errcheck — best-effort sidecar file, embedding already succeeded
	}

	return lyrics, nil
}

func (s *Server) handleFetchAndEmbedMultiple(c *fiber.Ctx) error {
	var req struct {
		FilePaths []string `json:"filePaths"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	results := make([]map[string]interface{}, len(req.FilePaths))
	for i, filePath := range req.FilePaths {
		result := map[string]interface{}{
			"filePath": filePath,
			"success":  false,
		}

		lyrics, err := s.fetchAndEmbedLyrics(filePath)
		if err != nil {
			result["error"] = err.Error()
		} else {
			result["success"] = true
			result["hasPlain"] = lyrics.Plain != ""
			result["hasSynced"] = lyrics.HasSynced
		}

		results[i] = result
	}

	return c.JSON(results)
}

// Qobuz handlers
func (s *Server) handleUpdateQobuzCredentials(c *fiber.Ctx) error {
	var req struct {
		AppID     string `json:"appId"`
		AppSecret string `json:"appSecret"`
		AuthToken string `json:"authToken"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	s.qobuzSource.SetCredentials(req.AppID, req.AppSecret, req.AuthToken)

	// Save to config
	s.config.QobuzAppID = req.AppID
	s.config.QobuzAppSecret = req.AppSecret
	s.config.QobuzAuthToken = req.AuthToken
	if err := core.SaveConfig(s.config); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

func (s *Server) handleIsQobuzConfigured(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"configured": s.qobuzSource.IsAvailable()})
}

// Folder handlers
func (s *Server) handleGetDownloadFolder(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"folder": s.config.DownloadFolder})
}

func (s *Server) handleSetDownloadFolder(c *fiber.Ctx) error {
	var req struct {
		Folder string `json:"folder"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	s.config.DownloadFolder = req.Folder
	if err := core.SaveConfig(s.config); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

// System handlers
func (s *Server) handleGetVersion(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"version": "1.0.0"})
}

func (s *Server) handleGetLogs(c *fiber.Ctx) error {
	// Implement log retrieval
	return c.JSON([]core.LogEntry{})
}

func (s *Server) handleClearLogs(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

func (s *Server) handleGetConnectionStatus(c *fiber.Ctx) error {
	// Check ffmpeg availability
	_, ffmpegErr := exec.LookPath("ffmpeg")
	return c.JSON(fiber.Map{
		"tidal":  s.tidalSource.IsAvailable(),
		"qobuz":  s.qobuzSource.IsAvailable(),
		"ffmpeg": ffmpegErr == nil,
	})
}

func (s *Server) handleIsDownloaderAvailable(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"available": s.tidalSource.GetService().IsAvailable()})
}
