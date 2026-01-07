package api

import (
	"os"
	"os/exec"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"flacidal/backend"
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
	var config backend.Config
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if err := backend.SaveConfig(&config); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	s.config = &config
	return c.JSON(fiber.Map{"success": true})
}

func (s *Server) handleResetConfig(c *fiber.Ctx) error {
	config := backend.GetDefaultConfig()
	if err := backend.SaveConfig(config); err != nil {
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

	source, err := s.sourceManager.DetectSource(req.URL)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Unknown URL format"})
	}

	id, contentType, err := source.ParseURL(req.URL)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
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
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		result["title"] = track.Title
		result["creator"] = track.Artist
		result["coverUrl"] = track.CoverURL
		result["tracks"] = []backend.SourceTrack{*track}

	case "album":
		album, err := source.GetAlbum(id)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		result["title"] = album.Title
		result["creator"] = album.Artist
		result["coverUrl"] = album.CoverURL
		result["tracks"] = album.Tracks

	case "playlist":
		playlist, err := source.GetPlaylist(id)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		result["title"] = playlist.Title
		result["creator"] = playlist.Creator
		result["coverUrl"] = playlist.CoverURL
		result["tracks"] = playlist.Tracks
	}

	return c.JSON(result)
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
	// Search not yet implemented for HTTP API
	return c.Status(501).JSON(fiber.Map{"error": "Search not implemented in server mode"})
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
		Tracks      []backend.TidalTrack `json:"tracks"`
		OutputDir   string               `json:"outputDir"`
		ContentName string               `json:"contentName"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	outputDir := req.OutputDir
	if outputDir == "" {
		outputDir = s.config.DownloadFolder
	}
	if outputDir == "" {
		outputDir = backend.GetDefaultDownloadFolder()
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
		outputDir = backend.GetDefaultDownloadFolder()
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
		"embedLyrics":     s.config.EmbedLyrics,
	})
}

func (s *Server) handleSetDownloadOptions(c *fiber.Ctx) error {
	var req struct {
		Quality         string `json:"quality"`
		FileNameFormat  string `json:"fileNameFormat"`
		OrganizeFolders bool   `json:"organizeFolders"`
		EmbedCover      bool   `json:"embedCover"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	s.config.DownloadQuality = req.Quality
	s.config.FileNameFormat = req.FileNameFormat
	s.config.OrganizeFolders = req.OrganizeFolders
	s.config.EmbedCover = req.EmbedCover

	if err := backend.SaveConfig(s.config); err != nil {
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
		outputDir = backend.GetDefaultDownloadFolder()
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
		return c.JSON([]backend.DownloadRecord{})
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
		return c.JSON(fiber.Map{"records": []backend.DownloadRecord{}, "total": 0})
	}

	filter := backend.HistoryFilter{
		Limit:  limit,
		Offset: offset,
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

func (s *Server) handleRefetchFromHistory(c *fiber.Ctx) error {
	// Implement refetch logic
	return c.JSON(fiber.Map{"error": "Not implemented"})
}

// Files handlers
func (s *Server) handleListFiles(c *fiber.Ctx) error {
	// File listing not yet implemented for HTTP API
	return c.Status(501).JSON(fiber.Map{"error": "File listing not implemented in server mode"})
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
	// Metadata reading not yet implemented for HTTP API
	return c.Status(501).JSON(fiber.Map{"error": "Metadata reading not implemented in server mode"})
}

func (s *Server) handleGetCoverArt(c *fiber.Ctx) error {
	// Cover art extraction not yet implemented for HTTP API
	return c.Status(501).JSON(fiber.Map{"error": "Cover art extraction not implemented in server mode"})
}

func (s *Server) handleGetRenameTemplates(c *fiber.Ctx) error {
	return c.JSON(backend.GetRenameTemplates())
}

func (s *Server) handlePreviewRename(c *fiber.Ctx) error {
	var req struct {
		Files    []string `json:"files"`
		Template string   `json:"template"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	previews := backend.PreviewRename(req.Files, req.Template)
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

	results := backend.RenameFiles(req.Files, req.Template)
	return c.JSON(results)
}

// Conversion handlers
func (s *Server) handleIsConverterAvailable(c *fiber.Ctx) error {
	// Check if ffmpeg is available by trying to run it
	_, err := exec.LookPath("ffmpeg")
	return c.JSON(fiber.Map{"available": err == nil})
}

func (s *Server) handleGetFFmpegInfo(c *fiber.Ctx) error {
	// FFmpeg info not yet implemented for HTTP API
	return c.Status(501).JSON(fiber.Map{"error": "FFmpeg info not implemented in server mode"})
}

func (s *Server) handleGetConversionFormats(c *fiber.Ctx) error {
	// Return standard conversion formats
	formats := []map[string]interface{}{
		{"id": "mp3", "name": "MP3", "extension": ".mp3"},
		{"id": "aac", "name": "AAC", "extension": ".m4a"},
		{"id": "ogg", "name": "OGG Vorbis", "extension": ".ogg"},
		{"id": "opus", "name": "Opus", "extension": ".opus"},
		{"id": "alac", "name": "ALAC", "extension": ".m4a"},
		{"id": "wav", "name": "WAV", "extension": ".wav"},
	}
	return c.JSON(formats)
}

func (s *Server) handleConvertFiles(c *fiber.Ctx) error {
	// File conversion not yet implemented for HTTP API
	return c.Status(501).JSON(fiber.Map{"error": "File conversion not implemented in server mode"})
}

// Analysis handlers
func (s *Server) handleAnalyzeFile(c *fiber.Ctx) error {
	// File analysis not yet implemented for HTTP API
	return c.Status(501).JSON(fiber.Map{"error": "File analysis not implemented in server mode"})
}

func (s *Server) handleAnalyzeMultiple(c *fiber.Ctx) error {
	// File analysis not yet implemented for HTTP API
	return c.Status(501).JSON(fiber.Map{"error": "File analysis not implemented in server mode"})
}

func (s *Server) handleQuickAnalyze(c *fiber.Ctx) error {
	// File analysis not yet implemented for HTTP API
	return c.Status(501).JSON(fiber.Map{"error": "File analysis not implemented in server mode"})
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
	// Lyrics for file not yet implemented for HTTP API (requires FLAC metadata reading)
	return c.Status(501).JSON(fiber.Map{"error": "Lyrics for file not implemented in server mode"})
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

	tagger := backend.NewFLACTagger()
	if err := tagger.EmbedLyrics(req.FilePath, req.Plain, req.Synced); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

func (s *Server) handleFetchAndEmbedLyrics(c *fiber.Ctx) error {
	// Lyrics fetch and embed not yet implemented for HTTP API (requires FLAC metadata reading)
	return c.Status(501).JSON(fiber.Map{"error": "Fetch and embed lyrics not implemented in server mode"})
}

func (s *Server) handleFetchAndEmbedMultiple(c *fiber.Ctx) error {
	// Lyrics fetch and embed not yet implemented for HTTP API (requires FLAC metadata reading)
	return c.Status(501).JSON(fiber.Map{"error": "Fetch and embed multiple lyrics not implemented in server mode"})
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
	if err := backend.SaveConfig(s.config); err != nil {
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
	if err := backend.SaveConfig(s.config); err != nil {
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
	return c.JSON([]backend.LogEntry{})
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
