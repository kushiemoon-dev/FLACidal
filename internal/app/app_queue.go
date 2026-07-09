package app

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	core "github.com/kushiemoon-dev/flacidal-core"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// =============================================================================
// Download Methods (exposed to frontend)
// =============================================================================

// OpenFLACFilesDialog opens a multi-file picker filtered to FLAC files.
func (a *App) OpenFLACFilesDialog() ([]string, error) {
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select FLAC Files to Analyze",
		Filters: []runtime.FileFilter{
			{DisplayName: "FLAC Audio (*.flac)", Pattern: "*.flac"},
		},
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

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
		a.config = &core.Config{}
	}
	a.config.DownloadFolder = folder
	return core.SaveConfig(a.config)
}

// IsDownloaderAvailable checks if the download service is reachable
func (a *App) IsDownloaderAvailable() bool {
	if a.downloader == nil {
		return false
	}
	return a.downloader.IsAvailable()
}

// DownloadTrack downloads a single track by its Tidal ID
func (a *App) DownloadTrack(trackID int, outputDir string) (*core.DownloadResult, error) {
	if a.downloader == nil {
		return nil, fmt.Errorf("downloader not initialized")
	}
	if outputDir == "" {
		return nil, fmt.Errorf("no output directory specified")
	}
	return a.downloader.DownloadTrack(trackID, outputDir, "", "", "", nil)
}

// DownloadTrackFromTidal downloads using TidalTrack data (for UI convenience)
func (a *App) DownloadTrackFromTidal(track core.TidalTrack, outputDir string) (*core.DownloadResult, error) {
	if a.downloader == nil {
		return nil, fmt.Errorf("downloader not initialized")
	}
	if outputDir == "" {
		return nil, fmt.Errorf("no output directory specified")
	}
	return a.downloader.DownloadTrack(track.ID, outputDir, track.Copyright, track.Label, "", nil)
}

// QueueDownloads queues multiple tracks for concurrent download
func (a *App) QueueDownloads(tracks []core.TidalTrack, outputDir string, contentName string, contentID string, contentType string) (int, error) {
	if a.downloadManager == nil {
		return 0, fmt.Errorf("download manager not initialized")
	}
	if outputDir == "" {
		return 0, fmt.Errorf("no output directory specified")
	}

	// Create subfolder with content name (playlist/album/track title)
	if contentName != "" {
		outputDir = filepath.Join(outputDir, core.SanitizeFileName(contentName))
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return 0, fmt.Errorf("failed to create folder: %w", err)
		}
	}

	queued := a.downloadManager.QueueMultiple(tracks, outputDir)

	// Save initial history record
	if a.db != nil && contentID != "" {
		if err := a.db.SaveDownloadRecord(&core.DownloadRecord{
			TidalContentID:   contentID,
			TidalContentName: contentName,
			ContentType:      contentType,
			TracksTotal:      queued,
		}); err != nil {
			a.logBuffer.Warn(fmt.Sprintf("Failed to save download history for %s: %v", contentID, err))
		}
	}
	// Map each trackID → contentID for progress callback
	for _, t := range tracks {
		a.trackContentMap.Store(t.ID, contentID)
	}

	return queued, nil
}

// QueueQobuzDownloads queues Qobuz-sourced tracks for concurrent download
func (a *App) QueueQobuzDownloads(tracks []core.SourceTrack, outputDir string, contentName string) (int, error) {
	if a.downloadManager == nil {
		return 0, fmt.Errorf("download manager not initialized")
	}
	if outputDir == "" {
		return 0, fmt.Errorf("no output directory specified")
	}
	if contentName != "" {
		outputDir = filepath.Join(outputDir, core.SanitizeFileName(contentName))
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return 0, fmt.Errorf("failed to create folder: %w", err)
		}
	}
	return a.downloadManager.QueueQobuzTracks(tracks, outputDir), nil
}

// QueueArtistAlbum fetches a Tidal album's tracks and queues them all for download.
// outputDir should be the artist folder; an album subfolder is created automatically.
func (a *App) QueueArtistAlbum(albumID string, artistName string, outputDir string) (int, error) {
	if a.downloadManager == nil {
		return 0, fmt.Errorf("download manager not initialized")
	}
	if outputDir == "" {
		return 0, fmt.Errorf("no output directory specified")
	}

	album, err := a.downloader.GetAlbumFromProxy(albumID)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch album: %w", err)
	}

	// Create {Artist}/{Album} folder structure
	artistFolder := core.SanitizeFileName(artistName)
	if artistFolder == "" {
		artistFolder = core.SanitizeFileName(album.Artist)
	}
	albumFolder := core.SanitizeFileName(album.Title)
	albumDir := filepath.Join(outputDir, artistFolder, albumFolder)
	if err := os.MkdirAll(albumDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create album folder: %w", err)
	}

	queued := a.downloadManager.QueueMultiple(album.Tracks, albumDir)
	return queued, nil
}

// DownloadArtistAssets downloads the artist's profile picture and banner image.
// Files are saved to {outputDir}/{artistName}/ as profile.jpg, profile_hires.jpg, banner.jpg.
// Returns the number of files successfully downloaded.
func (a *App) DownloadArtistAssets(artistID string, artistName string, outputDir string) (int, error) {
	if outputDir == "" {
		return 0, fmt.Errorf("no output directory specified")
	}

	name, pictureID, err := a.tidalClient.GetArtistPictureID(artistID)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch artist info: %w", err)
	}
	if pictureID == "" {
		return 0, fmt.Errorf("artist has no picture available")
	}

	// Use fetched name if caller didn't provide one
	if artistName == "" {
		artistName = name
	}

	urls := core.ArtistImageURLs(pictureID)
	if len(urls) == 0 {
		return 0, fmt.Errorf("no image URLs generated")
	}

	// Save to {outputDir}/{artistName}/
	destDir := filepath.Join(outputDir, core.SanitizeFileName(artistName))
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create artist folder: %w", err)
	}

	// Map of label → filename
	fileNames := map[string]string{
		"profile":       "profile.jpg",
		"profile_hires": "profile_hires.jpg",
		"banner":        "banner.jpg",
	}

	client := &http.Client{}
	downloaded := 0
	for label, imgURL := range urls {
		fname := fileNames[label]
		destPath := filepath.Join(destDir, fname)

		resp, err := client.Get(imgURL)
		if err != nil || resp.StatusCode != 200 {
			if resp != nil {
				resp.Body.Close()
			}
			continue // skip unavailable sizes
		}

		f, err := os.Create(destPath)
		if err != nil {
			resp.Body.Close()
			continue
		}
		_, copyErr := io.Copy(f, resp.Body)
		f.Close()
		resp.Body.Close()
		if copyErr == nil {
			downloaded++
		}
	}

	return downloaded, nil
}

// QueueSingleDownload queues a single track for download
func (a *App) QueueSingleDownload(trackID int, outputDir, title, artist string) error {
	if a.downloadManager == nil {
		return fmt.Errorf("download manager not initialized")
	}
	if outputDir == "" {
		return fmt.Errorf("no output directory specified")
	}

	// Fetch ISRC from Tidal metadata so the orchestrator can search by ISRC on fallback sources.
	isrc := ""
	if track, err := a.downloader.GetTrackAsTidalTrack(trackID); err == nil && track != nil {
		isrc = track.ISRC
		if title == "" {
			title = track.Title
		}
		if artist == "" {
			artist = track.Artist
		}
	}

	err := a.downloadManager.QueueDownloadWithISRC(trackID, outputDir, title, artist, isrc)
	if err == nil && a.db != nil {
		contentID := strconv.Itoa(trackID)
		if saveErr := a.db.SaveDownloadRecord(&core.DownloadRecord{
			TidalContentID:   contentID,
			TidalContentName: title,
			ContentType:      "track",
			TracksTotal:      1,
		}); saveErr != nil {
			a.logBuffer.Warn(fmt.Sprintf("Failed to save download history for %s: %v", contentID, saveErr))
		}
		a.trackContentMap.Store(trackID, contentID)
	}
	return err
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
			"saveCoverFile":   true,
			"saveFolderCover": true,
			"autoAnalyze":     false,
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
		"saveCoverFile":   a.config.SaveCoverFile,
		"saveFolderCover": a.config.SaveFolderCover,
		"autoAnalyze":     a.config.AutoAnalyze,
	}
}

// SetDownloadOptions updates download options
func (a *App) SetDownloadOptions(quality, fileNameFormat string, organizeFolders, embedCover, saveCoverFile, autoAnalyze bool) error {
	if a.config == nil {
		a.config = &core.Config{}
	}

	a.config.DownloadQuality = quality
	a.config.FileNameFormat = fileNameFormat
	a.config.OrganizeFolders = organizeFolders
	a.config.EmbedCover = embedCover
	a.config.SaveCoverFile = saveCoverFile
	a.config.AutoAnalyze = autoAnalyze

	// Update downloader options (preserve AutoQualityFallback from config)
	if a.downloader != nil {
		autoQualityFallback := false
		if a.config != nil {
			autoQualityFallback = a.config.AutoQualityFallback
		}
		a.downloader.SetOptions(core.DownloadOptions{
			Quality:             quality,
			FileNameFormat:      fileNameFormat,
			OrganizeFolders:     organizeFolders,
			EmbedCover:          embedCover,
			SaveCoverFile:       saveCoverFile,
			AutoAnalyze:         autoAnalyze,
			AutoQualityFallback: autoQualityFallback,
		})
	}

	return core.SaveConfig(a.config)
}

// OpenDownloadFolder opens the download folder in the system file manager
func (a *App) OpenDownloadFolder(folder string) error {
	if folder == "" {
		return fmt.Errorf("no folder specified")
	}
	runtime.BrowserOpenURL(a.ctx, "file://"+folder)
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

	isrc, title, artist := "", "", ""
	if track, err := a.downloader.GetTrackAsTidalTrack(trackID); err == nil && track != nil {
		isrc, title, artist = track.ISRC, track.Title, track.Artist
	}
	return a.downloadManager.QueueDownloadWithISRC(trackID, folder, title, artist, isrc)
}

// RetryAllFailed retries all failed downloads
func (a *App) RetryAllFailed() (int, error) {
	if a.downloadManager == nil {
		return 0, fmt.Errorf("download manager not initialized")
	}

	count := a.downloadManager.RetryAllFailed()
	return count, nil
}

// ExportFailedDownloads saves failed download info to a TXT or CSV file chosen by the user.
// format is "txt" or "csv". Returns the path of the saved file, or empty string if cancelled.
func (a *App) ExportFailedDownloads(format string) (string, error) {
	if a.downloadManager == nil {
		return "", fmt.Errorf("download manager not initialized")
	}
	jobs := a.downloadManager.GetFailedJobs()
	if len(jobs) == 0 {
		return "", nil
	}

	// Determine file filter and default name
	var filter []runtime.FileFilter
	var defaultFilename string
	if format == "csv" {
		filter = []runtime.FileFilter{{DisplayName: "CSV Files", Pattern: "*.csv"}}
		defaultFilename = "failed_downloads.csv"
	} else {
		filter = []runtime.FileFilter{{DisplayName: "Text Files", Pattern: "*.txt"}}
		defaultFilename = "failed_downloads.txt"
	}

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: defaultFilename,
		Filters:         filter,
	})
	if err != nil || savePath == "" {
		return "", err
	}

	var sb strings.Builder
	if format == "csv" {
		sb.WriteString("artist,title,url,error\n")
		for _, job := range jobs {
			url := fmt.Sprintf("https://tidal.com/browse/track/%d", job.TrackID)
			sb.WriteString(fmt.Sprintf("%q,%q,%q,%q\n", job.Artist, job.Title, url, job.Error))
		}
	} else {
		for _, job := range jobs {
			url := fmt.Sprintf("https://tidal.com/browse/track/%d", job.TrackID)
			sb.WriteString(fmt.Sprintf("%s - %s | %s | %s\n", job.Artist, job.Title, url, job.Error))
		}
	}

	if err := os.WriteFile(savePath, []byte(sb.String()), 0644); err != nil {
		return "", err
	}
	return savePath, nil
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
