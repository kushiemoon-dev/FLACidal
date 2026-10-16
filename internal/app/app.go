package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sync"
	"time"

	core "github.com/kushiemoon-dev/flacidal-core"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	version         string
	core            *core.Core // owner of the config and of every component below once Startup ran
	ctx             context.Context
	configMu        sync.RWMutex // guards config, see cfg/setCfg/updateConfig
	config          *core.Config
	db              *core.Database
	tidalClient     *core.TidalClient
	spotifySearch   *core.SpotifyClient // Client Credentials, so no login is required
	matcher         *core.Matcher
	downloader      *core.TidalHifiService
	downloadManager *core.DownloadManager
	logBuffer       *core.LogBuffer
	sourceManager   *core.SourceManager
	tidalSource     *core.TidalSource
	qobuzSource     *core.QobuzSource
	amazonSource    *core.AmazonSource
	soulseekSource  *core.SoulseekSource
	deezerSource    *core.DeezerSource         // metadata only
	spotifySource   *core.SpotifySource        // metadata only
	bandcampSource  *core.BandcampSource       // pay-what-you-want downloads
	orchestrator    *core.DownloadOrchestrator // applies live priority changes
	trackContentMap sync.Map                   // tracks trackID -> contentID, used when recording history
}

func NewApp(version string) *App {
	return &App{version: version}
}

func DefaultSldlPath() string {
	if goruntime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData, _ = os.UserConfigDir()
		}
		return filepath.Join(appData, "flacidal", "sldl.exe")
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".local", "share", "flacidal", "sldl")
}

// EnsureSldlExecutable makes sure the sldl binary can run and isn't quarantined by the OS.
// On Linux and macOS this sets the executable bit, the same way the FFmpeg installer does.
// On macOS it additionally strips the com.apple.quarantine xattr that Gatekeeper attaches to
// browser-downloaded files; skipping this causes the process to be killed at launch even
// when os.Stat confirms the file exists.
func EnsureSldlExecutable(path string) error {
	if goruntime.GOOS == "windows" {
		return nil
	}
	if err := os.Chmod(path, 0755); err != nil {
		return fmt.Errorf("failed to chmod +x sldl: %w", err)
	}
	if goruntime.GOOS == "darwin" {
		exec.Command("xattr", "-d", "com.apple.quarantine", path).Run() //nolint:errcheck // attr commonly absent, not an error
	}
	return nil
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

	// Only backs the Soulseek status and install screens; downloads use Core's native client.
	cfg := c.Config()
	a.soulseekSource = NewSldlSource(&cfg, a.logBuffer)
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	c, err := core.New("", core.Options{InitEndpoints: core.InitTidalEndpoints})
	if err != nil {
		log.Printf("Could not start the core: %v", err)
		return
	}
	a.attachCore(c)

	// A single-file event channel keeps ExecuteJS calls from overlapping, which otherwise crashes WebKit on Linux.
	type progressEvent struct {
		eventType string // defaults to "download-progress" when left blank
		trackID   int
		status    string
		result    *core.DownloadResult
	}
	eventCh := make(chan progressEvent, 64)
	go func() {
		for ev := range eventCh {
			evType := ev.eventType
			if evType == "" {
				evType = "download-progress"
			}
			runtime.EventsEmit(ctx, evType, map[string]interface{}{
				"trackId": ev.trackID,
				"status":  ev.status,
				"result":  ev.result,
			})
			// Brief pause between events so WebKit/GTK can process the JS
			time.Sleep(50 * time.Millisecond)
		}
	}()

	a.downloadManager.SetProgressCallback(func(trackID int, status string, result *core.DownloadResult) {
		if a.logBuffer != nil {
			switch status {
			case "queued":
				a.logBuffer.Info(fmt.Sprintf("Queued track %d", trackID))
			case "downloading":
				a.logBuffer.Info(fmt.Sprintf("Now downloading track %d...", trackID))
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
						eventCh <- progressEvent{eventType: "endpoint-cooldown", trackID: -1, status: "cooldown", result: &core.DownloadResult{
							Error: fmt.Sprintf("all endpoints cooling down, resuming in %ds", minCooldown),
						}}
					}
				}
			case "cancelled":
				a.logBuffer.Warn(fmt.Sprintf("Cancelled track %d", trackID))
			}
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

		// Blocks briefly if the buffer is full, trivial next to download time.
		eventCh <- progressEvent{trackID: trackID, status: status, result: result}
	})
	// Starts the workers and restores the persisted queue; the callback above is already in place.
	a.core.Start()
	a.logBuffer.Success("FLACidal is ready!")
}

func (a *App) Shutdown(ctx context.Context) {
	if a.core != nil {
		a.core.Shutdown()
	}

	if a.configLoaded() {
		cfg := a.cfg()
		core.SaveConfig(&cfg)
	}
}
