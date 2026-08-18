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

func (a *App) SelectDownloadFolder() (string, error) {
	folder, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Download Folder",
	})
	if err != nil {
		return "", err
	}
	return folder, nil
}

func (a *App) GetDownloadFolder() string {
	if a.config != nil && a.config.DownloadFolder != "" {
		return a.config.DownloadFolder
	}
	return ""
}

func (a *App) SetDownloadFolder(folder string) error {
	if a.config == nil {
		a.config = &core.Config{}
	}
	a.config.DownloadFolder = folder
	return core.SaveConfig(a.config)
}

func (a *App) IsDownloaderAvailable() bool {
	if a.downloader == nil {
		return false
	}
	return a.downloader.IsAvailable()
}

func (a *App) DownloadTrack(trackID int, outputDir string) (*core.DownloadResult, error) {
	if a.downloader == nil {
		return nil, fmt.Errorf("downloader not initialized")
	}
	if outputDir == "" {
		return nil, fmt.Errorf("no output directory specified")
	}
	return a.downloader.DownloadTrack(trackID, outputDir, "", "", "", nil)
}

func (a *App) DownloadTrackFromTidal(track core.TidalTrack, outputDir string) (*core.DownloadResult, error) {
	if a.downloader == nil {
		return nil, fmt.Errorf("downloader not initialized")
	}
	if outputDir == "" {
		return nil, fmt.Errorf("no output directory specified")
	}
	return a.downloader.DownloadTrack(track.ID, outputDir, track.Copyright, track.Label, "", nil)
}

func (a *App) QueueDownloads(tracks []core.TidalTrack, outputDir string, contentName string, contentID string, contentType string) (int, error) {
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

	queued := a.downloadManager.QueueMultiple(tracks, outputDir)

	if a.db != nil && contentID != "" {
		if err := a.db.SaveDownloadRecord(&core.DownloadRecord{
			TidalContentID:   contentID,
			TidalContentName: contentName,
			ContentType:      contentType,
			TracksTotal:      queued,
		}); err != nil {
			a.logBuffer.Warn(fmt.Sprintf("Could not save download history for %s: %v", contentID, err))
		}
	}
	// Consumed by the progress callback to attribute a trackID back to its content.
	for _, t := range tracks {
		a.trackContentMap.Store(t.ID, contentID)
	}

	return queued, nil
}

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

// outputDir should point at the artist's folder; an album subfolder gets created automatically.
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

// Files are written to {outputDir}/{artistName}/ as profile.jpg, profile_hires.jpg, banner.jpg.
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

	if artistName == "" {
		artistName = name
	}

	urls := core.ArtistImageURLs(pictureID)
	if len(urls) == 0 {
		return 0, fmt.Errorf("no image URLs generated")
	}

	destDir := filepath.Join(outputDir, core.SanitizeFileName(artistName))
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create artist folder: %w", err)
	}

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
			continue // this size isn't available, move on
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

func (a *App) QueueSingleDownload(trackID int, outputDir, title, artist string) error {
	if a.downloadManager == nil {
		return fmt.Errorf("download manager not initialized")
	}
	if outputDir == "" {
		return fmt.Errorf("no output directory specified")
	}

	// Pull the full Tidal metadata, not just the ISRC, so sources that don't
	// self-tag (see needsRetag in flacidal-core) still end up with
	// album/tracknumber/discnumber/year/cover embedded post-download.
	var err error
	track, lookupErr := a.downloader.GetTrackAsTidalTrack(trackID)
	if lookupErr == nil && track != nil {
		if title != "" {
			track.Title = title
		}
		if artist != "" {
			track.Artist = artist
		}
		title, artist = track.Title, track.Artist
		err = a.downloadManager.QueueDownloadTrack(*track, outputDir)
	} else {
		err = a.downloadManager.QueueDownloadWithISRC(trackID, outputDir, title, artist, "")
	}
	if err == nil && a.db != nil {
		contentID := strconv.Itoa(trackID)
		if saveErr := a.db.SaveDownloadRecord(&core.DownloadRecord{
			TidalContentID:   contentID,
			TidalContentName: title,
			ContentType:      "track",
			TracksTotal:      1,
		}); saveErr != nil {
			a.logBuffer.Warn(fmt.Sprintf("Could not save download history for %s: %v", contentID, saveErr))
		}
		a.trackContentMap.Store(trackID, contentID)
	}
	return err
}

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

	// AutoQualityFallback isn't a SetDownloadOptions parameter, so preserve it from the existing config.
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

func (a *App) OpenDownloadFolder(folder string) error {
	if folder == "" {
		return fmt.Errorf("no folder specified")
	}
	runtime.BrowserOpenURL(a.ctx, "file://"+folder)
	return nil
}

func (a *App) RetryDownload(trackID int) error {
	if a.downloadManager == nil {
		return fmt.Errorf("download manager not initialized")
	}

	folder := a.GetDownloadFolder()
	if folder == "" {
		return fmt.Errorf("no download folder configured")
	}

	if track, err := a.downloader.GetTrackAsTidalTrack(trackID); err == nil && track != nil {
		return a.downloadManager.QueueDownloadTrack(*track, folder)
	}
	return a.downloadManager.QueueDownloadWithISRC(trackID, folder, "", "", "")
}

func (a *App) RetryAllFailed() (int, error) {
	if a.downloadManager == nil {
		return 0, fmt.Errorf("download manager not initialized")
	}

	count := a.downloadManager.RetryAllFailed()
	return count, nil
}

// format should be "txt" or "csv". It returns the saved file's path, or an empty string if the user cancelled.
func (a *App) ExportFailedDownloads(format string) (string, error) {
	if a.downloadManager == nil {
		return "", fmt.Errorf("download manager not initialized")
	}
	jobs := a.downloadManager.GetFailedJobs()
	if len(jobs) == 0 {
		return "", nil
	}

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

func (a *App) CancelDownload(trackID int) error {
	if a.downloadManager == nil {
		return fmt.Errorf("download manager not initialized")
	}

	return a.downloadManager.CancelDownload(trackID)
}

func (a *App) PauseDownloads() bool {
	if a.downloadManager == nil {
		return false
	}

	success := a.downloadManager.PauseQueue()
	if success && a.logBuffer != nil {
		a.logBuffer.Info("Paused the download queue")
		runtime.EventsEmit(a.ctx, "queue-paused", true)
	}
	return success
}

func (a *App) ResumeDownloads() bool {
	if a.downloadManager == nil {
		return false
	}

	success := a.downloadManager.ResumeQueue()
	if success && a.logBuffer != nil {
		a.logBuffer.Info("Resumed the download queue")
		runtime.EventsEmit(a.ctx, "queue-paused", false)
	}
	return success
}

func (a *App) IsQueuePaused() bool {
	if a.downloadManager == nil {
		return false
	}
	return a.downloadManager.IsPaused()
}
