package backend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// m3u8Entry holds data for one line in a generated M3U8 playlist.
type m3u8Entry struct {
	duration int
	artist   string
	title    string
	relPath  string
}

// m3u8Batch tracks progress of a download batch for M3U8 generation.
type m3u8Batch struct {
	total   int
	done    int
	entries []m3u8Entry
}

// DownloadManager handles concurrent downloads with queue
type DownloadManager struct {
	service         *TidalHifiService
	qobuzSource     *QobuzSource // optional fallback source
	sourceOrder     []string     // e.g. ["tidal", "qobuz"]
	workers         int
	queue           chan *DownloadJob
	results         chan *DownloadResult
	activeJobs      map[int]*DownloadJob
	failedJobs      map[int]*DownloadJob // Track failed jobs for retry
	mu              sync.RWMutex
	wg              sync.WaitGroup
	running         bool
	paused          bool       // Pause state
	pauseCond       *sync.Cond // Condition variable for pause/resume
	onProgress      func(trackID int, status string, result *DownloadResult)
	generateM3U8    bool
	batches         map[string]*m3u8Batch
	batchMu         sync.Mutex
	skipUnavailable bool
}

// DownloadJob represents a single download task
type DownloadJob struct {
	TrackID    int                `json:"trackId"`
	OutputDir  string             `json:"outputDir"`
	Title      string             `json:"title"`
	Artist     string             `json:"artist"`
	ISRC       string             `json:"isrc,omitempty"`      // Used for cross-source fallback matching
	Duration   int                `json:"duration,omitempty"`  // Track duration in seconds (for M3U8)
	Error      string             `json:"error,omitempty"`     // Error message if download failed
	Copyright  string             `json:"copyright,omitempty"` // Copyright string to embed in tags
	Label      string             `json:"label,omitempty"`     // Record label to embed in tags
	Source     string             `json:"source,omitempty"`    // "tidal" or "qobuz" — routes to correct downloader
	ctx        context.Context    // For cancellation
	cancelFunc context.CancelFunc // Cancel function
}

// DownloadProgress represents download progress for frontend
type DownloadProgress struct {
	TrackID  int    `json:"trackId"`
	Status   string `json:"status"`   // "queued", "downloading", "completed", "error"
	Progress int    `json:"progress"` // 0-100
	Error    string `json:"error,omitempty"`
	FileSize int64  `json:"fileSize,omitempty"`
	FilePath string `json:"filePath,omitempty"`
}

// DownloadEvent represents a download event for WebSocket broadcasts
type DownloadEvent struct {
	TrackID int             `json:"trackId"`
	Status  string          `json:"status"` // "queued", "downloading", "completed", "error", "cancelled"
	Result  *DownloadResult `json:"result,omitempty"`
}

// NewDownloadManager creates a new download manager
func NewDownloadManager(service *TidalHifiService, workers int) *DownloadManager {
	if workers <= 0 {
		workers = 3 // Default concurrent downloads
	}
	if workers > 10 {
		workers = 10 // Max limit
	}

	dm := &DownloadManager{
		service:     service,
		sourceOrder: []string{"tidal"},
		workers:     workers,
		queue:       make(chan *DownloadJob, 1000), // Large buffer for big playlists
		results:     make(chan *DownloadResult, 1000),
		activeJobs:  make(map[int]*DownloadJob),
		failedJobs:  make(map[int]*DownloadJob),
		batches:     make(map[string]*m3u8Batch),
	}
	dm.pauseCond = sync.NewCond(&dm.mu)
	return dm
}

// SetFallbackQobuzSource sets the Qobuz source to use when Tidal fails.
func (dm *DownloadManager) SetFallbackQobuzSource(source *QobuzSource) {
	dm.qobuzSource = source
}

// SetSourceOrder sets the priority order for sources, e.g. ["tidal", "qobuz"].
func (dm *DownloadManager) SetSourceOrder(order []string) {
	if len(order) > 0 {
		dm.sourceOrder = order
	}
}

// SetProgressCallback sets the callback for progress updates
func (dm *DownloadManager) SetProgressCallback(callback func(trackID int, status string, result *DownloadResult)) {
	dm.onProgress = callback
}

// SetGenerateM3U8 enables or disables automatic M3U8 generation after batch downloads.
func (dm *DownloadManager) SetGenerateM3U8(enabled bool) {
	dm.generateM3U8 = enabled
}

// SetSkipUnavailable configures whether unavailable tracks are silently skipped.
func (dm *DownloadManager) SetSkipUnavailable(skip bool) {
	dm.skipUnavailable = skip
}

// Start begins the worker pool
func (dm *DownloadManager) Start() {
	dm.mu.Lock()
	if dm.running {
		dm.mu.Unlock()
		return
	}
	dm.running = true
	dm.mu.Unlock()

	// Start worker goroutines
	for i := 0; i < dm.workers; i++ {
		dm.wg.Add(1)
		go dm.worker(i)
	}
}

// Stop gracefully stops the download manager
func (dm *DownloadManager) Stop() {
	dm.mu.Lock()
	if !dm.running {
		dm.mu.Unlock()
		return
	}
	dm.running = false
	dm.paused = false
	dm.pauseCond.Broadcast() // Wake up any paused workers so they can exit
	dm.mu.Unlock()

	close(dm.queue)
	dm.wg.Wait()
}

// worker processes download jobs from the queue
func (dm *DownloadManager) worker(id int) {
	defer dm.wg.Done()

	for job := range dm.queue {
		// Wait if paused
		dm.mu.Lock()
		for dm.paused && dm.running {
			dm.pauseCond.Wait()
		}
		dm.mu.Unlock()

		// Check if still running after waiting
		dm.mu.RLock()
		running := dm.running
		dm.mu.RUnlock()
		if !running {
			return
		}

		dm.processJob(job)
	}
}

// processJob downloads a single track
func (dm *DownloadManager) processJob(job *DownloadJob) {
	// Check if already cancelled before starting
	if job.ctx != nil {
		select {
		case <-job.ctx.Done():
			if dm.onProgress != nil {
				dm.onProgress(job.TrackID, "cancelled", nil)
			}
			return
		default:
		}
	}

	// Notify start
	if dm.onProgress != nil {
		dm.onProgress(job.TrackID, "downloading", nil)
	}

	// Mark as active
	dm.mu.Lock()
	dm.activeJobs[job.TrackID] = job
	// Remove from failed if retrying
	delete(dm.failedJobs, job.TrackID)
	dm.mu.Unlock()

	// Download — route by source
	var result *DownloadResult
	var err error
	if job.Source == "qobuz" && dm.qobuzSource != nil && dm.qobuzSource.IsAvailable() {
		opts := dm.service.GetOptions()
		result, err = dm.qobuzSource.DownloadTrack(
			strconv.Itoa(job.TrackID), job.OutputDir, opts,
		)
		if result != nil {
			result.Source = "qobuz"
		}
	} else {
		result, err = dm.service.DownloadTrack(job.TrackID, job.OutputDir, job.Copyright, job.Label)
		if (err != nil || (result != nil && !result.Success)) && dm.qobuzFallbackEnabled() && job.ISRC != "" {
			result, err = dm.downloadViaQobuzFallback(job, result)
		}
	}

	// Check for cancellation after download
	cancelled := false
	if job.ctx != nil {
		select {
		case <-job.ctx.Done():
			cancelled = true
		default:
		}
	}

	// Remove from active
	dm.mu.Lock()
	delete(dm.activeJobs, job.TrackID)
	dm.mu.Unlock()

	// Handle result
	if cancelled {
		if dm.onProgress != nil {
			dm.onProgress(job.TrackID, "cancelled", nil)
		}
	} else if err != nil || !result.Success {
		// Capture error message for export
		if result != nil && result.Error != "" {
			job.Error = result.Error
		} else if err != nil {
			job.Error = err.Error()
		}
		// Track failed job for retry
		dm.mu.Lock()
		dm.failedJobs[job.TrackID] = job
		dm.mu.Unlock()

		if dm.onProgress != nil {
			dm.onProgress(job.TrackID, "error", result)
		}
	} else if dm.onProgress != nil {
		dm.onProgress(job.TrackID, "completed", result)
	}

	// Send to results channel (non-blocking)
	select {
	case dm.results <- result:
	default:
	}

	// Record for M3U8 batch tracking
	if dm.generateM3U8 && !cancelled {
		dm.recordBatchResult(job, result)
	}
}

// recordBatchResult updates batch progress and writes M3U8 when batch completes.
func (dm *DownloadManager) recordBatchResult(job *DownloadJob, result *DownloadResult) {
	dm.batchMu.Lock()
	batch, ok := dm.batches[job.OutputDir]
	if !ok {
		dm.batchMu.Unlock()
		return
	}
	batch.done++
	if result != nil && result.Success && result.FilePath != "" {
		relPath, err := filepath.Rel(job.OutputDir, result.FilePath)
		if err != nil {
			relPath = filepath.Base(result.FilePath)
		}
		batch.entries = append(batch.entries, m3u8Entry{
			duration: job.Duration,
			artist:   job.Artist,
			title:    job.Title,
			relPath:  relPath,
		})
	}
	if batch.done < batch.total {
		dm.batchMu.Unlock()
		return
	}
	// All jobs finished — take entries and clean up
	entries := batch.entries
	delete(dm.batches, job.OutputDir)
	dm.batchMu.Unlock()

	if len(entries) > 0 {
		dm.writeM3U8(job.OutputDir, entries)
	}
}

// writeM3U8 writes a .m3u8 playlist file into outputDir.
func (dm *DownloadManager) writeM3U8(outputDir string, entries []m3u8Entry) {
	name := filepath.Base(outputDir)
	outPath := filepath.Join(outputDir, name+".m3u8")

	var sb strings.Builder
	sb.WriteString("#EXTM3U\n")
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("#EXTINF:%d,%s - %s\n", e.duration, e.artist, e.title))
		sb.WriteString(e.relPath + "\n")
	}

	_ = os.WriteFile(outPath, []byte(sb.String()), 0644)
	if dm.service != nil && dm.service.logger != nil {
		dm.service.logger.Info(fmt.Sprintf("M3U8 playlist written: %s", outPath))
	}
}

// qobuzFallbackEnabled returns true when Qobuz fallback is configured and available.
func (dm *DownloadManager) qobuzFallbackEnabled() bool {
	if dm.qobuzSource == nil || !dm.qobuzSource.IsAvailable() {
		return false
	}
	for _, s := range dm.sourceOrder {
		if s == "qobuz" {
			return true
		}
	}
	return false
}

// downloadViaQobuzFallback attempts to download a track via Qobuz when Tidal fails.
// It uses the ISRC for matching, falling back to title+artist search.
func (dm *DownloadManager) downloadViaQobuzFallback(job *DownloadJob, tidalResult *DownloadResult) (*DownloadResult, error) {
	logger := dm.service.logger
	if logger != nil {
		logger.Warn(fmt.Sprintf("Tidal failed for '%s - %s', falling back to Qobuz", job.Artist, job.Title))
	}

	// Find the track on Qobuz
	var qTrack *SourceTrack
	var err error
	if job.ISRC != "" {
		qTrack, _ = dm.qobuzSource.SearchTrackByISRC(job.ISRC)
	}
	if qTrack == nil {
		qTrack, err = dm.qobuzSource.SearchTrackByTitleArtist(job.Title, job.Artist)
		if err != nil {
			if logger != nil {
				logger.Error(fmt.Sprintf("Qobuz fallback: track not found ('%s - %s'): %v", job.Artist, job.Title, err))
			}
			return tidalResult, err
		}
	}

	// Download via Qobuz
	opts := dm.service.GetOptions()
	result, err := dm.qobuzSource.DownloadTrack(qTrack.ID, job.OutputDir, opts)
	if err != nil {
		if logger != nil {
			logger.Error(fmt.Sprintf("Qobuz fallback download failed: %v", err))
		}
		return tidalResult, err
	}

	result.Source = "qobuz"
	if logger != nil {
		logger.Info(fmt.Sprintf("Qobuz fallback succeeded: %s", result.FilePath))
	}
	return result, nil
}

// QueueDownload adds a track to the download queue
func (dm *DownloadManager) QueueDownload(trackID int, outputDir, title, artist string) error {
	return dm.QueueDownloadWithISRC(trackID, outputDir, title, artist, "")
}

// QueueDownloadWithISRC adds a track with ISRC metadata (used for cross-source fallback).
func (dm *DownloadManager) QueueDownloadWithISRC(trackID int, outputDir, title, artist, isrc string) error {
	return dm.queueDownloadFull(trackID, outputDir, title, artist, isrc, 0, "", "")
}

// queueDownloadFull is the internal full-parameter queuing function.
func (dm *DownloadManager) queueDownloadFull(trackID int, outputDir, title, artist, isrc string, duration int, copyright, label string) error {
	dm.mu.RLock()
	if !dm.running {
		dm.mu.RUnlock()
		return fmt.Errorf("download manager not running")
	}
	dm.mu.RUnlock()

	// Create context for cancellation
	ctx, cancelFunc := context.WithCancel(context.Background())

	job := &DownloadJob{
		TrackID:    trackID,
		OutputDir:  outputDir,
		Title:      title,
		Artist:     artist,
		ISRC:       isrc,
		Duration:   duration,
		Copyright:  copyright,
		Label:      label,
		ctx:        ctx,
		cancelFunc: cancelFunc,
	}

	// Add to queue (blocking - will wait if queue is full)
	dm.queue <- job

	// Notify queued only after successfully added
	if dm.onProgress != nil {
		dm.onProgress(trackID, "queued", nil)
	}

	return nil
}

// QueueMultiple adds multiple tracks to the queue
func (dm *DownloadManager) QueueMultiple(tracks []TidalTrack, outputDir string) int {
	queued := 0
	for _, track := range tracks {
		// Skip unavailable tracks if configured
		if dm.skipUnavailable && !track.Available {
			continue
		}
		err := dm.queueDownloadFull(track.ID, outputDir, track.Title, track.Artist, track.ISRC, track.Duration, track.Copyright, track.Label)
		if err == nil {
			queued++
		}
	}
	// Register batch for M3U8 generation (only when at least one track was queued)
	if dm.generateM3U8 && queued > 0 {
		dm.batchMu.Lock()
		dm.batches[outputDir] = &m3u8Batch{total: queued}
		dm.batchMu.Unlock()
	}
	return queued
}

// GetActiveCount returns the number of currently downloading tracks
func (dm *DownloadManager) GetActiveCount() int {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return len(dm.activeJobs)
}

// GetQueueLength returns the number of tracks waiting in queue
func (dm *DownloadManager) GetQueueLength() int {
	return len(dm.queue)
}

// IsRunning returns whether the download manager is active
func (dm *DownloadManager) IsRunning() bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.running
}

// CancelDownload cancels a download in progress
func (dm *DownloadManager) CancelDownload(trackID int) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Check if job is active
	job, exists := dm.activeJobs[trackID]
	if !exists {
		return fmt.Errorf("track %d is not currently downloading", trackID)
	}

	// Cancel the context
	if job.cancelFunc != nil {
		job.cancelFunc()
	}

	return nil
}

// RetryAllFailed re-queues all failed downloads
func (dm *DownloadManager) RetryAllFailed() int {
	dm.mu.Lock()
	// Copy failed jobs to avoid holding lock during queue operations
	jobsToRetry := make([]*DownloadJob, 0, len(dm.failedJobs))
	for _, job := range dm.failedJobs {
		jobsToRetry = append(jobsToRetry, job)
	}
	// Clear failed jobs map
	dm.failedJobs = make(map[int]*DownloadJob)
	dm.mu.Unlock()

	// Re-queue each failed job
	retried := 0
	for _, job := range jobsToRetry {
		err := dm.QueueDownloadWithISRC(job.TrackID, job.OutputDir, job.Title, job.Artist, job.ISRC)
		if err == nil {
			retried++
		}
	}

	return retried
}

// GetFailedCount returns the number of failed downloads
func (dm *DownloadManager) GetFailedCount() int {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return len(dm.failedJobs)
}

// GetFailedJobs returns a snapshot of all failed jobs (for export purposes).
func (dm *DownloadManager) GetFailedJobs() []DownloadJob {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	jobs := make([]DownloadJob, 0, len(dm.failedJobs))
	for _, job := range dm.failedJobs {
		jobs = append(jobs, *job)
	}
	return jobs
}

// ClearFailed removes all failed jobs from tracking
func (dm *DownloadManager) ClearFailed() int {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	count := len(dm.failedJobs)
	dm.failedJobs = make(map[int]*DownloadJob)
	return count
}

// PauseQueue pauses the download queue (active downloads continue, new ones wait)
func (dm *DownloadManager) PauseQueue() bool {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if dm.paused {
		return false // Already paused
	}
	dm.paused = true
	return true
}

// ResumeQueue resumes the download queue
func (dm *DownloadManager) ResumeQueue() bool {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if !dm.paused {
		return false // Already running
	}
	dm.paused = false
	dm.pauseCond.Broadcast() // Wake up all waiting workers
	return true
}

// IsPaused returns whether the queue is paused
func (dm *DownloadManager) IsPaused() bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.paused
}

// QueueQobuzTracks queues Qobuz-sourced tracks for download via QobuzSource.
func (dm *DownloadManager) QueueQobuzTracks(tracks []SourceTrack, outputDir string) int {
	dm.mu.RLock()
	if !dm.running {
		dm.mu.RUnlock()
		return 0
	}
	dm.mu.RUnlock()

	queued := 0
	for _, t := range tracks {
		trackID, err := strconv.Atoi(t.ID)
		if err != nil {
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		job := &DownloadJob{
			TrackID:    trackID,
			OutputDir:  outputDir,
			Title:      t.Title,
			Artist:     t.Artist,
			ISRC:       t.ISRC,
			Duration:   t.Duration,
			Source:     "qobuz",
			ctx:        ctx,
			cancelFunc: cancel,
		}
		dm.queue <- job
		if dm.onProgress != nil {
			dm.onProgress(trackID, "queued", &DownloadResult{TrackID: trackID, Title: t.Title, Artist: t.Artist})
		}
		queued++
	}
	if dm.generateM3U8 && queued > 0 {
		dm.batchMu.Lock()
		dm.batches[outputDir] = &m3u8Batch{total: queued}
		dm.batchMu.Unlock()
	}
	return queued
}
