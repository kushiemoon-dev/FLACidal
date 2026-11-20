package app

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	core "github.com/kushiemoon-dev/flacidal-core"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	version           string
	core              *core.Core // owner of the config and of every component below once Startup ran
	ctx               context.Context
	configMu          sync.RWMutex // guards config, see cfg/setCfg/updateConfig
	config            *core.Config
	db                *core.Database
	tidalClient       *core.TidalClient
	spotifySearch     *core.SpotifyClient // Client Credentials, so no login is required
	matcher           *core.Matcher
	downloader        *core.TidalHifiService
	downloadManager   *core.DownloadManager
	analyzeBatches    BatchRegistry
	logBuffer         *core.LogBuffer
	sourceManager     *core.SourceManager
	tidalSource       *core.TidalSource
	ffmpeg            FFmpegInstaller
	emit              func(event string, data any) // Wails event sink, injectable in tests
	cancelLogs        func()
	events            *serialEmitter // single path to the front end, see emitEvent
	qobuzSource       *core.QobuzSource
	amazonSource      *core.AmazonSource
	deezerSource      *core.DeezerSource         // metadata only
	spotifySource     *core.SpotifySource        // metadata only
	bandcampSource    *core.BandcampSource       // pay-what-you-want downloads
	orchestrator      *core.DownloadOrchestrator // applies live priority changes
	trackContentMap   sync.Map                   // tracks trackID -> contentID, used when recording history
	downloadingLogged sync.Map                   // trackIDs whose "Now downloading" line was already logged
}

func NewApp(version string) *App {
	return &App{version: version}
}

// attachCore points the App's component fields at the ones Core built. It is
// the only place the App learns about them, so desktop and headless share one wiring.
func (a *App) attachCore(c *core.Core) {
	a.core = c
	a.logBuffer = c.LogBuffer()
	a.db = c.DB()
	a.tidalClient = c.TidalClient()
	a.spotifySearch = c.SpotifySearch()
	a.matcher = c.Matcher()
	a.downloader = c.Downloader()
	a.downloadManager = c.DownloadManager()
	a.sourceManager = c.SourceManager()
	a.tidalSource = c.TidalSource()
	a.qobuzSource = c.QobuzSource()
	a.amazonSource = c.AmazonSource()
	a.deezerSource = c.DeezerSource()
	a.spotifySource = c.SpotifySource()
	a.bandcampSource = c.BandcampSource()
	a.orchestrator = c.Orchestrator()
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	c, err := core.New("", core.Options{InitEndpoints: core.InitTidalEndpoints})
	if err != nil {
		log.Printf("Could not start the core: %v", err)
		return
	}
	a.attachCore(c)
	// One goroutine for every front end event, so ExecuteJS calls never overlap
	// (log lines included), which otherwise crashes WebKit on Linux.
	events := newSerialEmitter(func(name string, data any) { runtime.EventsEmit(ctx, name, data) }, 50*time.Millisecond)
	a.events = events
	a.emit = events.TrySend
	a.wireLogEvents()
	sendProgress := func(eventType string, trackID int, status string, result *core.DownloadResult) {
		payload := map[string]interface{}{
			"trackId": trackID,
			"status":  status,
			"result":  result,
		}
		if status == "downloading" {
			// Progress ticks arrive several times a second per download and only
			// the latest one matters.
			events.SendCoalesced(fmt.Sprintf("track:%d", trackID), eventType, payload)
			return
		}
		events.Send(eventType, payload)
	}

	a.downloadManager.SetProgressCallback(func(trackID int, status string, result *core.DownloadResult) {
		if a.logBuffer != nil {
			switch status {
			case "queued":
				a.logBuffer.Info(fmt.Sprintf("Queued track %d", trackID))
			case "downloading":
				a.logDownloading(trackID)
			case "completed":
				if result != nil {
					a.logBuffer.Success(fmt.Sprintf("Download complete: %s (quality: %s)", result.FilePath, result.Quality))
					if result.QualityMismatch {
						a.logBuffer.Warn(fmt.Sprintf("Quality mismatch, asked for %s, received %s",
							result.RequestedQuality, result.Quality))
					}
					if result.Analysis != nil {
						if result.Analysis.IsTrueLossless {
							a.logBuffer.Info(fmt.Sprintf("Analysis: %s, confirmed true lossless", result.Analysis.VerdictLabel))
						} else {
							a.logBuffer.Warn(fmt.Sprintf("Analysis: %s, possibly upscaled from a lossy source", result.Analysis.VerdictLabel))
						}
					}
				}
			case "error":
				if result != nil && result.Error != "" {
					a.logBuffer.Error(fmt.Sprintf("Download error: %s", result.Error))
				}
				if a.cfg().AutoStopOnCooldown && !a.downloader.HasHealthyEndpoints() {
					if a.downloadManager.PauseQueue() {
						a.logBuffer.Warn("Every Tidal endpoint is cooling down, queue paused")
						minCooldown := 0
						for _, stat := range a.downloader.PoolSnapshot() {
							if stat.CooldownSecs > 0 && (minCooldown == 0 || stat.CooldownSecs < minCooldown) {
								minCooldown = stat.CooldownSecs
							}
						}
						sendProgress("endpoint-cooldown", -1, "cooldown", &core.DownloadResult{
							Error: fmt.Sprintf("all endpoints cooling down, resuming in %ds", minCooldown),
						})
					}
				}
			case "cancelled":
				a.logBuffer.Warn(fmt.Sprintf("Cancelled track %d", trackID))
			}
		}

		switch status {
		case "completed", "error", "cancelled":
			a.clearDownloading(trackID)
		}

		if a.db != nil {
			switch status {
			case "completed":
				if cid, ok := a.trackContentMap.Load(trackID); ok {
					if err := a.db.IncrementDownloadCounts(cid.(string), true); err != nil {
						a.logBuffer.Warn(fmt.Sprintf("Could not update download counts for %s: %v", cid.(string), err))
					}
					a.trackContentMap.Delete(trackID)
				}
			case "error":
				if cid, ok := a.trackContentMap.Load(trackID); ok {
					if err := a.db.IncrementDownloadCounts(cid.(string), false); err != nil {
						a.logBuffer.Warn(fmt.Sprintf("Could not update download counts for %s: %v", cid.(string), err))
					}
					a.trackContentMap.Delete(trackID)
				}
			}
		}

		sendProgress("download-progress", trackID, status, result)
	})
	// Starts the workers and restores the persisted queue; the callback above is already in place.
	a.core.Start()
	a.logBuffer.Success("FLACidal is ready!")
}

func (a *App) Shutdown(ctx context.Context) {
	if a.cancelLogs != nil {
		a.cancelLogs()
	}
	if a.core != nil {
		a.core.Shutdown()
	}
	if a.events != nil {
		a.events.Close()
	}

	if a.configLoaded() {
		cfg := a.cfg()
		core.SaveConfig(&cfg)
	}
}
