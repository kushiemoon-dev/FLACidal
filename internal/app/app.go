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

// App struct - main Wails application
type App struct {
	version         string
	ctx             context.Context
	config          *core.Config
	db              *core.Database
	tidalClient     *core.TidalClient
	spotifySearch   *core.SpotifyClient // For search/matching (Client Credentials, no login)
	matcher         *core.Matcher
	downloader      *core.TidalHifiService     // FLAC downloader
	downloadManager *core.DownloadManager      // Concurrent download manager
	logBuffer       *core.LogBuffer            // Log buffer for Terminal page
	sourceManager   *core.SourceManager        // Multi-source manager
	tidalSource     *core.TidalSource          // Tidal source
	qobuzSource     *core.QobuzSource          // Qobuz source
	amazonSource    *core.AmazonSource         // Amazon Music fallback source
	soulseekSource  *core.SoulseekSource       // Soulseek last-resort source
	deezerSource    *core.DeezerSource         // Deezer metadata-only source
	spotifySource   *core.SpotifySource        // Spotify metadata-only source
	bandcampSource  *core.BandcampSource       // Bandcamp name-your-price source
	orchestrator    *core.DownloadOrchestrator // Download orchestrator for live priority updates
	trackContentMap sync.Map                   // maps trackID (int) → contentID (string) for history tracking
}

// NewApp creates a new App application struct
func NewApp(version string) *App {
	return &App{version: version}
}

// defaultSldlPath returns the platform-appropriate default path for the sldl binary.
func defaultSldlPath() string {
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

// ensureSldlExecutable ensures the sldl binary is executable and not quarantined.
// On Linux/macOS it sets the executable bit (mirrors what the FFmpeg installer does).
// On macOS it also removes the com.apple.quarantine xattr that Gatekeeper applies to
// files downloaded via a browser — without this the process is killed on launch even
// though os.Stat reports the file as present.
func ensureSldlExecutable(path string) error {
	if goruntime.GOOS == "windows" {
		return nil
	}
	if err := os.Chmod(path, 0755); err != nil {
		return fmt.Errorf("chmod +x on sldl failed: %w", err)
	}
	if goruntime.GOOS == "darwin" {
		exec.Command("xattr", "-d", "com.apple.quarantine", path).Run() //nolint:errcheck // attr commonly absent, not an error
	}
	return nil
}

// Startup is called when the app starts
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize log buffer
	a.logBuffer = core.NewLogBuffer(500)
	a.logBuffer.Info("FLACidal starting...")

	// Start background fetch of dynamic Tidal endpoints (before downloader is created).
	core.SetTidalEndpointLogger(a.logBuffer)
	core.InitTidalEndpoints()

	// Load config
	config, err := core.LoadConfig()
	if err != nil {
		a.logBuffer.Warn("Could not load config: " + err.Error())
		config = &core.Config{}
	}
	a.config = config
	a.logBuffer.Success("Configuration loaded")

	// Initialize database
	db, err := core.NewDatabase()
	if err != nil {
		a.logBuffer.Error("Database initialization failed: " + err.Error())
	} else {
		a.logBuffer.Success("Database initialized")
	}
	a.db = db

	// Initialize Tidal client (uses internal credentials, no user config needed)
	a.tidalClient = core.NewTidalClientDefault()
	a.tidalClient.SetCountryCode(config.CountryCode)
	if config.ProxyURL != "" {
		if err := a.tidalClient.SetProxy(config.ProxyURL); err != nil {
			a.logBuffer.Warn("Proxy config error (Tidal API): " + err.Error())
		} else {
			a.logBuffer.Info("Tidal API proxy: " + config.ProxyURL)
		}
	}
	a.logBuffer.Info("Tidal client ready")

	// Initialize Spotify search client (Client Credentials, no login needed)
	a.spotifySearch = core.NewSpotifyClientForSearch()

	// Initialize matcher
	a.matcher = core.NewMatcher(a.spotifySearch, a.db)

	// Initialize FLAC downloader
	a.downloader = core.NewTidalHifiService()
	// Attach logger so endpoint rotation events appear in Terminal page
	a.downloader.SetLogger(a.logBuffer)
	if config.ProxyURL != "" {
		if err := a.downloader.SetProxy(config.ProxyURL); err != nil {
			a.logBuffer.Warn("Proxy config error (downloader): " + err.Error())
		}
	}
	// Apply custom endpoints if configured.
	// TidalHifiEndpoints = total override (no gist); TidalCustomEndpoint = prepend to dynamic list.
	if len(config.TidalHifiEndpoints) > 0 {
		a.downloader.SetEndpoints(config.TidalHifiEndpoints)
		a.logBuffer.Info(fmt.Sprintf("Tidal HiFi endpoint pool: %d endpoints configured (override)", len(config.TidalHifiEndpoints)))
	} else {
		base := core.GetTidalEndpoints()
		priority := config.TidalPriorityEndpoints
		if len(priority) == 0 && config.TidalCustomEndpoint != "" {
			priority = []string{config.TidalCustomEndpoint} // backward compat with legacy single-endpoint field
		}
		if len(priority) > 0 {
			a.downloader.SetEndpoints(append(priority, base...))
			a.logBuffer.Info(fmt.Sprintf("Tidal HiFi priority endpoints: %d self-host + %d public", len(priority), len(base)))
		}
	}
	// Set download options from config
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
	a.logBuffer.Info("FLAC downloader service ready")

	// Initialize download manager with 4 concurrent workers
	a.downloadManager = core.NewDownloadManager(a.downloader, 4)
	a.downloadManager.SetJellyfin(config.JellyfinEnabled, config.JellyfinURL, config.JellyfinAPIKey)

	// Serialized event channel to avoid concurrent ExecuteJS calls that crash WebKit on Linux.
	// Events are queued and emitted one at a time from a dedicated goroutine.
	type progressEvent struct {
		eventType string // "download-progress" if empty
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
			// Small delay between events to let WebKit/GTK process JS
			time.Sleep(50 * time.Millisecond)
		}
	}()

	a.downloadManager.SetProgressCallback(func(trackID int, status string, result *core.DownloadResult) {
		// Log download events
		if a.logBuffer != nil {
			switch status {
			case "queued":
				a.logBuffer.Info(fmt.Sprintf("Track %d added to queue", trackID))
			case "downloading":
				a.logBuffer.Info(fmt.Sprintf("Downloading track %d...", trackID))
			case "completed":
				if result != nil {
					a.logBuffer.Success(fmt.Sprintf("Downloaded: %s (quality: %s)", result.FilePath, result.Quality))
					if result.QualityMismatch {
						a.logBuffer.Warn(fmt.Sprintf("Quality mismatch: requested %s but got %s",
							result.RequestedQuality, result.Quality))
					}
					if result.Analysis != nil {
						if result.Analysis.IsTrueLossless {
							a.logBuffer.Info(fmt.Sprintf("Analysis: %s - True lossless", result.Analysis.VerdictLabel))
						} else {
							a.logBuffer.Warn(fmt.Sprintf("Analysis: %s - May be upscaled from lossy source", result.Analysis.VerdictLabel))
						}
					}
				}
			case "error":
				if result != nil && result.Error != "" {
					a.logBuffer.Error(fmt.Sprintf("Download failed: %s", result.Error))
				}
				// Auto-stop if all endpoints are in cooldown and the feature is enabled
				if a.config.AutoStopOnCooldown && !a.downloader.HasHealthyEndpoints() {
					if a.downloadManager.PauseQueue() {
						a.logBuffer.Warn("All Tidal endpoints in cooldown — queue paused")
						// Find the minimum cooldown across all dead endpoints
						minCooldown := 0
						for _, stat := range a.downloader.PoolSnapshot() {
							if stat.CooldownSecs > 0 && (minCooldown == 0 || stat.CooldownSecs < minCooldown) {
								minCooldown = stat.CooldownSecs
							}
						}
						eventCh <- progressEvent{eventType: "endpoint-cooldown", trackID: -1, status: "cooldown", result: &core.DownloadResult{
							Error: fmt.Sprintf("all endpoints in cooldown, resuming in %ds", minCooldown),
						}}
					}
				}
			case "cancelled":
				a.logBuffer.Warn(fmt.Sprintf("Track %d cancelled", trackID))
			}
		}

		// Update download history counts
		if a.db != nil {
			switch status {
			case "completed":
				if cid, ok := a.trackContentMap.Load(trackID); ok {
					if err := a.db.IncrementDownloadCounts(cid.(string), true); err != nil {
						a.logBuffer.Warn(fmt.Sprintf("Failed to update download counts for %s: %v", cid.(string), err))
					}
					a.trackContentMap.Delete(trackID)
				}
			case "error":
				if cid, ok := a.trackContentMap.Load(trackID); ok {
					if err := a.db.IncrementDownloadCounts(cid.(string), false); err != nil {
						a.logBuffer.Warn(fmt.Sprintf("Failed to update download counts for %s: %v", cid.(string), err))
					}
					a.trackContentMap.Delete(trackID)
				}
			}
		}

		// Queue event for serialized emission (blocking — workers wait
		// briefly if buffer is full, which is negligible vs download time)
		eventCh <- progressEvent{trackID: trackID, status: status, result: result}
	})
	a.downloadManager.Start()
	a.logBuffer.Success("Download manager started (4 workers)")

	// Initialize source manager
	a.sourceManager = core.NewSourceManager()

	// Initialize Tidal source
	a.tidalSource = core.NewTidalSource()
	a.tidalSource.SetAvailable(config.TidalEnabled)
	a.sourceManager.RegisterSource(a.tidalSource)
	a.logBuffer.Info("Tidal source registered")

	// Initialize Qobuz source
	a.qobuzSource = core.NewQobuzSource(config.QobuzAppID, config.QobuzAppSecret)
	a.qobuzSource.SetLogger(a.logBuffer)
	if config.ProxyURL != "" {
		if err := a.qobuzSource.SetProxy(config.ProxyURL); err != nil {
			a.logBuffer.Warn("Proxy config error (Qobuz): " + err.Error())
		}
	}
	if len(config.QobuzEndpoints) > 0 {
		a.qobuzSource.SetEndpoints(config.QobuzEndpoints)
		a.logBuffer.Info(fmt.Sprintf("Qobuz endpoint pool: %d endpoints configured (override)", len(config.QobuzEndpoints)))
	} else {
		base := core.DefaultQobuzEndpoints()
		priority := config.QobuzPriorityEndpoints
		if len(priority) == 0 && config.QobuzCustomEndpoint != "" {
			priority = []string{config.QobuzCustomEndpoint} // backward compat with legacy single-endpoint field
		}
		if len(priority) > 0 {
			a.qobuzSource.SetEndpoints(append(priority, base...))
			a.logBuffer.Info(fmt.Sprintf("Qobuz priority endpoints: %d self-host + %d public", len(priority), len(base)))
		}
	}
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

	// Configure inter-source fallback for download manager
	if config.QobuzEnabled && a.qobuzSource.IsAvailable() {
		a.downloadManager.SetFallbackQobuzSource(a.qobuzSource)
	}
	// Circuit breaker: wire TidalSource so selectBestService can check endpoint health
	a.downloadManager.SetTidalSource(a.tidalSource)

	// Initialize Amazon Music fallback source (no auth required, via proxy pool)
	a.amazonSource = core.NewAmazonSource()
	if len(config.AmazonProxyEndpoints) > 0 {
		a.amazonSource.SetEndpoints(config.AmazonProxyEndpoints)
		a.logBuffer.Info(fmt.Sprintf("Amazon endpoint pool: %d endpoints configured (override)", len(config.AmazonProxyEndpoints)))
	} else if priority := config.AmazonPriorityEndpoints; len(priority) > 0 {
		base := core.GetEndpoints("amazon")
		a.amazonSource.SetEndpoints(append(priority, base...))
		a.logBuffer.Info(fmt.Sprintf("Amazon priority endpoints: %d self-host + %d public", len(priority), len(base)))
	}
	a.sourceManager.RegisterSource(a.amazonSource)
	a.logBuffer.Info("Amazon Music fallback source initialized")

	// Initialize Deezer and Spotify metadata-only sources (URL routing, no download)
	a.deezerSource = core.NewDeezerSource()
	a.sourceManager.RegisterSource(a.deezerSource)
	a.spotifySource = core.NewSpotifySource(a.spotifySearch)
	a.sourceManager.RegisterSource(a.spotifySource)

	// Initialize Bandcamp source (name-your-price / free FLAC downloads)
	a.bandcampSource = core.NewBandcampSource()
	a.sourceManager.RegisterSource(a.bandcampSource)
	a.logBuffer.Info("Bandcamp source initialized")

	// Initialize Soulseek fallback source (last-resort P2P, independent of streaming proxies)
	sldlPath := config.SoulseekBinaryPath
	if sldlPath == "" {
		sldlPath = defaultSldlPath()
	}
	if err := ensureSldlExecutable(sldlPath); err != nil {
		a.logBuffer.Warn(fmt.Sprintf("sldl binary may not be executable: %v", err))
	}
	a.soulseekSource = core.NewSoulseekSource(sldlPath, config.SoulseekUsername, config.SoulseekPassword)
	a.soulseekSource.SetLogger(a.logBuffer)
	if config.SoulseekEnabled && a.soulseekSource.IsAvailable() {
		a.sourceManager.RegisterSource(a.soulseekSource)
		a.logBuffer.Info("Soulseek fallback source initialized")
	} else if config.SoulseekEnabled {
		if config.SoulseekBinaryPath != "" {
			if _, err := os.Stat(config.SoulseekBinaryPath); os.IsNotExist(err) {
				a.logBuffer.Warn(fmt.Sprintf("Soulseek enabled but binary not found at %s", config.SoulseekBinaryPath))
			}
		} else if _, err := os.Stat(sldlPath); os.IsNotExist(err) {
			a.logBuffer.Warn(fmt.Sprintf("Soulseek enabled but binary not found at default path %s", sldlPath))
		}
		if config.SoulseekUsername == "" || config.SoulseekPassword == "" {
			a.logBuffer.Warn("Soulseek enabled but username/password not configured")
		}
	}

	// Wire multi-source orchestrator, priority order shared with downloadManager below
	// (previously a separate hardcoded list that ignored config.SourceOrder and put
	// Soulseek last, contradicting the Soulseek-first default described in the README).
	sourceOrder := config.SourceOrder
	if len(sourceOrder) == 0 {
		sourceOrder = core.DefaultSourceOrder(config)
	}
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

	a.logBuffer.Success("FLACidal ready!")
}

// Shutdown is called when the app is closing
func (a *App) Shutdown(ctx context.Context) {
	// Stop download manager
	if a.downloadManager != nil {
		a.downloadManager.Stop()
	}

	// Save config
	if a.config != nil {
		core.SaveConfig(a.config)
	}

	// Close database
	if a.db != nil {
		a.db.Close()
	}
}
