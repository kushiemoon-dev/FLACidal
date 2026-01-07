package backend

import (
	"context"
	"fmt"
	"sync"
)

// DownloadManager handles concurrent downloads with queue
type DownloadManager struct {
	service     *TidalHifiService
	workers     int
	queue       chan *DownloadJob
	results     chan *DownloadResult
	activeJobs  map[int]*DownloadJob
	failedJobs  map[int]*DownloadJob // Track failed jobs for retry
	mu          sync.RWMutex
	wg          sync.WaitGroup
	running     bool
	paused      bool        // Pause state
	pauseCond   *sync.Cond  // Condition variable for pause/resume
	onProgress  func(trackID int, status string, result *DownloadResult)
}

// DownloadJob represents a single download task
type DownloadJob struct {
	TrackID    int                `json:"trackId"`
	OutputDir  string             `json:"outputDir"`
	Title      string             `json:"title"`
	Artist     string             `json:"artist"`
	ctx        context.Context    // For cancellation
	cancelFunc context.CancelFunc // Cancel function
}

// DownloadProgress represents download progress for frontend
type DownloadProgress struct {
	TrackID  int    `json:"trackId"`
	Status   string `json:"status"` // "queued", "downloading", "completed", "error"
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
		service:    service,
		workers:    workers,
		queue:      make(chan *DownloadJob, 1000), // Large buffer for big playlists
		results:    make(chan *DownloadResult, 1000),
		activeJobs: make(map[int]*DownloadJob),
		failedJobs: make(map[int]*DownloadJob),
	}
	dm.pauseCond = sync.NewCond(&dm.mu)
	return dm
}

// SetProgressCallback sets the callback for progress updates
func (dm *DownloadManager) SetProgressCallback(callback func(trackID int, status string, result *DownloadResult)) {
	dm.onProgress = callback
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

	// Download
	result, err := dm.service.DownloadTrack(job.TrackID, job.OutputDir)

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
		// Track failed job for retry
		dm.mu.Lock()
		dm.failedJobs[job.TrackID] = job
		dm.mu.Unlock()

		if dm.onProgress != nil {
			dm.onProgress(job.TrackID, "error", result)
		}
	} else {
		if dm.onProgress != nil {
			dm.onProgress(job.TrackID, "completed", result)
		}
	}

	// Send to results channel (non-blocking)
	select {
	case dm.results <- result:
	default:
	}
}

// QueueDownload adds a track to the download queue
func (dm *DownloadManager) QueueDownload(trackID int, outputDir, title, artist string) error {
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
		err := dm.QueueDownload(track.ID, outputDir, track.Title, track.Artist)
		if err == nil {
			queued++
		}
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
		err := dm.QueueDownload(job.TrackID, job.OutputDir, job.Title, job.Artist)
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
