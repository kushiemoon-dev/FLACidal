package app

import (
	"context"
	"fmt"
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
	ctx             context.Context
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

// resolveAndPersistSourceOrder returns config.SourceOrder, computing it with
// core.DefaultSourceOrder and writing the result back to config whenever it
// was blank — this way App.GetConfig() (and by extension Settings.svelte)
// later reflects the order the orchestrator actually applies, rather than
// leaving that resolution stuck in a local variable that never hits disk.
// warnf is given a best-effort diagnostic when the persist step fails; pass
// nil if logging isn't needed.
func resolveAndPersistSourceOrder(config *core.Config, warnf func(string)) []string {
	if len(config.SourceOrder) > 0 {
		return config.SourceOrder
	}
	config.SourceOrder = core.DefaultSourceOrder(config)
	if err := core.SaveConfig(config); err != nil && warnf != nil {
		warnf(fmt.Sprintf("could not persist resolved source order: %v", err))
	}
	return config.SourceOrder
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	a.logBuffer = core.NewLogBuffer(500)
	a.logBuffer.Info("Starting FLACidal...")

	// Must run before the downloader is constructed further down, which reads these endpoints.
	core.SetTidalEndpointLogger(a.logBuffer)
	core.InitTidalEndpoints()

	config, err := core.LoadConfig()
	if err != nil {
		a.logBuffer.Warn("Unable to load config: " + err.Error())
		config = &core.Config{}
	}
	a.config = config
	a.logBuffer.Success("Config loaded successfully")

	db, err := core.NewDatabase()
	if err != nil {
		a.logBuffer.Error("Database init failed: " + err.Error())
	} else {
		a.logBuffer.Success("Database ready")
	}
	a.db = db

	// Relies on built-in credentials — no user configuration is needed.
	a.tidalClient = core.NewTidalClientDefault()
	a.tidalClient.SetCountryCode(config.CountryCode)
	if config.ProxyURL != "" {
		if err := a.tidalClient.SetProxy(config.ProxyURL); err != nil {
			a.logBuffer.Warn("Tidal API proxy misconfigured: " + err.Error())
		} else {
			a.logBuffer.Info("Using Tidal API proxy: " + config.ProxyURL)
		}
	}
	a.logBuffer.Info("Tidal client is ready")

	// Client Credentials — no login needed.
	a.spotifySearch = core.NewSpotifyClientForSearch()

	a.matcher = core.NewMatcher(a.spotifySearch, a.db)

	a.downloader = core.NewTidalHifiService()
	// So endpoint rotation shows up on the Terminal page.
	a.downloader.SetLogger(a.logBuffer)
	if config.ProxyURL != "" {
		if err := a.downloader.SetProxy(config.ProxyURL); err != nil {
			a.logBuffer.Warn("Downloader proxy misconfigured: " + err.Error())
		}
	}
	// TidalHifiEndpoints fully replaces the default endpoints.
	if len(config.TidalHifiEndpoints) > 0 {
		a.downloader.SetEndpoints(config.TidalHifiEndpoints)
		a.logBuffer.Info(fmt.Sprintf("Tidal HiFi endpoints: %d configured (override)", len(config.TidalHifiEndpoints)))
	} else {
		base := core.GetTidalEndpoints()
		a.downloader.SetEndpoints(base)
	}
	quality := config.DownloadQuality
	if quality == "" {
		quality = "LOSSLESS"
	}
	fileNameFormat := config.FileNameFormat
	if fileNameFormat == "" {
		fileNameFormat = "{artist} - {title}"
	}
	a.downloader.SetOptions(core.DownloadOptions{
		Quality:              quality,
		FileNameFormat:       fileNameFormat,
		OrganizeFolders:      config.OrganizeFolders,
		FolderTemplate:       config.FolderTemplate,
		EmbedCover:           config.EmbedCover,
		SaveCoverFile:        config.SaveCoverFile,
		AutoAnalyze:          config.AutoAnalyze,
		AutoQualityFallback:  config.AutoQualityFallback,
		QualityFallbackOrder: config.QualityOrder,
		FirstArtistOnly:      config.FirstArtistOnly,
		SkipExisting:         config.SkipExisting,
		ExternalLibraryPaths: config.ExternalLibraryPaths,
		ArtistSeparator:      config.ArtistSeparator,
		PlaylistSubfolder:    config.PlaylistSubfolder,
		SaveLyricsFile:       config.SaveLyricsFile,
		SaveFolderCover:      config.SaveFolderCover,
	})
	a.logBuffer.Info("FLAC downloader ready")

	a.downloadManager = core.NewDownloadManager(a.downloader, 4)
	a.downloadManager.SetJellyfin(config.JellyfinEnabled, config.JellyfinURL, config.JellyfinAPIKey)
	if a.db != nil {
		a.downloadManager.SetJobCompleteCallback(func(entry core.HistoryEntry) {
			if err := a.db.InsertHistoryEntry(entry); err != nil {
				a.logBuffer.Warn(fmt.Sprintf("Could not save per-track history for '%s - %s': %v", entry.Artist, entry.Title, err))
			}
		})
	}

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
						a.logBuffer.Warn(fmt.Sprintf("Quality mismatch — asked for %s, received %s",
							result.RequestedQuality, result.Quality))
					}
					if result.Analysis != nil {
						if result.Analysis.IsTrueLossless {
							a.logBuffer.Info(fmt.Sprintf("Analysis: %s — confirmed true lossless", result.Analysis.VerdictLabel))
						} else {
							a.logBuffer.Warn(fmt.Sprintf("Analysis: %s — possibly upscaled from a lossy source", result.Analysis.VerdictLabel))
						}
					}
				}
			case "error":
				if result != nil && result.Error != "" {
					a.logBuffer.Error(fmt.Sprintf("Download error: %s", result.Error))
				}
				if a.config.AutoStopOnCooldown && !a.downloader.HasHealthyEndpoints() {
					if a.downloadManager.PauseQueue() {
						a.logBuffer.Warn("Every Tidal endpoint is cooling down — queue paused")
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

		// Blocks briefly if the buffer is full — trivial next to download time.
		eventCh <- progressEvent{trackID: trackID, status: status, result: result}
	})
	a.downloadManager.Start()
	a.logBuffer.Success("Download manager running (4 workers)")

	a.sourceManager = core.NewSourceManager()

	a.tidalSource = core.NewTidalSource()
	a.tidalSource.SetAvailable(config.TidalEnabled)
	if len(config.TidalHifiEndpoints) > 0 {
		a.tidalSource.GetService().SetEndpoints(config.TidalHifiEndpoints)
	} else {
		base := core.GetTidalEndpoints()
		a.tidalSource.GetService().SetEndpoints(base)
	}
	a.sourceManager.RegisterSource(a.tidalSource)
	a.logBuffer.Info("Registered the Tidal source")

	a.qobuzSource = core.NewQobuzSource(config.QobuzAppID, config.QobuzAppSecret)
	a.qobuzSource.SetLogger(a.logBuffer)
	if config.ProxyURL != "" {
		if err := a.qobuzSource.SetProxy(config.ProxyURL); err != nil {
			a.logBuffer.Warn("Qobuz proxy misconfigured: " + err.Error())
		}
	}
	if len(config.QobuzEndpoints) > 0 {
		a.qobuzSource.SetEndpoints(config.QobuzEndpoints)
		a.logBuffer.Info(fmt.Sprintf("Qobuz endpoints: %d configured (override)", len(config.QobuzEndpoints)))
	} else {
		base := core.DefaultQobuzEndpoints()
		a.qobuzSource.SetEndpoints(base)
	}
	if config.QobuzAuthToken != "" {
		a.qobuzSource.SetCredentials(config.QobuzAppID, config.QobuzAppSecret, config.QobuzAuthToken)
	}
	a.sourceManager.RegisterSource(a.qobuzSource)
	if config.QobuzEnabled && config.QobuzAppID != "" {
		a.logBuffer.Info("Registered the Qobuz source")
	}

	if config.PreferredSource != "" {
		a.sourceManager.SetPreferredSource(config.PreferredSource)
	}

	if config.QobuzEnabled && a.qobuzSource.IsAvailable() {
		a.downloadManager.SetFallbackQobuzSource(a.qobuzSource)
	}
	// Circuit breaker: give the download manager the TidalSource so selectBestService can check endpoint health
	a.downloadManager.SetTidalSource(a.tidalSource)

	// No auth needed; routed through the proxy pool.
	a.amazonSource = core.NewAmazonSource()
	if len(config.AmazonProxyEndpoints) > 0 {
		a.amazonSource.SetEndpoints(config.AmazonProxyEndpoints)
		a.logBuffer.Info(fmt.Sprintf("Amazon endpoints: %d configured (override)", len(config.AmazonProxyEndpoints)))
	} else {
		base := core.GetEndpoints("amazon")
		a.amazonSource.SetEndpoints(base)
	}
	a.sourceManager.RegisterSource(a.amazonSource)
	a.logBuffer.Info("Amazon Music fallback source ready")

	// Used for URL routing rather than downloads.
	a.deezerSource = core.NewDeezerSource()
	a.sourceManager.RegisterSource(a.deezerSource)
	a.spotifySource = core.NewSpotifySource(a.spotifySearch)
	a.sourceManager.RegisterSource(a.spotifySource)

	a.bandcampSource = core.NewBandcampSource()
	a.sourceManager.RegisterSource(a.bandcampSource)
	a.logBuffer.Info("Bandcamp source ready")

	// A last-resort P2P path, independent of the streaming proxies.
	sldlPath := config.SoulseekBinaryPath
	if sldlPath == "" {
		sldlPath = DefaultSldlPath()
	}
	if err := EnsureSldlExecutable(sldlPath); err != nil {
		a.logBuffer.Warn(fmt.Sprintf("sldl binary might not be runnable: %v", err))
	}
	a.soulseekSource = core.NewSoulseekSource(sldlPath, config.SoulseekUsername, config.SoulseekPassword)
	a.soulseekSource.SetLogger(a.logBuffer)
	if config.SoulseekEnabled && a.soulseekSource.IsAvailable() {
		a.sourceManager.RegisterSource(a.soulseekSource)
		a.logBuffer.Info("Soulseek fallback source ready")
	} else if config.SoulseekEnabled {
		if config.SoulseekBinaryPath != "" {
			if _, err := os.Stat(config.SoulseekBinaryPath); os.IsNotExist(err) {
				a.logBuffer.Warn(fmt.Sprintf("Soulseek is enabled but no binary was found at %s", config.SoulseekBinaryPath))
			}
		} else if _, err := os.Stat(sldlPath); os.IsNotExist(err) {
			a.logBuffer.Warn(fmt.Sprintf("Soulseek is enabled but no binary exists at the default path %s", sldlPath))
		}
		if config.SoulseekUsername == "" || config.SoulseekPassword == "" {
			a.logBuffer.Warn("Soulseek is enabled but no username/password is set")
		}
	}

	// Wire up the multi-source orchestrator; the priority order is shared with
	// downloadManager below (this used to be a separate hardcoded list that ignored
	// config.SourceOrder and placed Soulseek last, which contradicted the
	// Soulseek-first default the README describes).
	sourceOrder := resolveAndPersistSourceOrder(config, func(msg string) { a.logBuffer.Warn(msg) })
	a.orchestrator = core.NewDownloadOrchestrator(a.sourceManager, sourceOrder, a.logBuffer)
	if a.db != nil {
		a.orchestrator.SetDatabase(a.db)
		if a.soulseekSource != nil {
			a.soulseekSource.SetDatabase(a.db)
		}
	}
	a.downloadManager.SetOrchestrator(a.orchestrator)
	a.downloadManager.SetSourceOrder(sourceOrder)
	a.downloadManager.SetGenerateM3U8(config.GenerateM3U8)
	a.downloadManager.SetSkipUnavailable(config.SkipUnavailableTracks)

	a.logBuffer.Success("FLACidal is ready!")
}

func (a *App) Shutdown(ctx context.Context) {
	if a.downloadManager != nil {
		a.downloadManager.Stop()
	}

	if a.config != nil {
		core.SaveConfig(a.config)
	}

	if a.db != nil {
		a.db.Close()
	}
}
