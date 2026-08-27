package main

import (
	"context"
	"embed"
	"log"
	"os"
	"os/signal"
	"syscall"

	"flacidal/internal/api"
	"flacidal/internal/app"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// frontendFS stays empty by default; the server falls back to serving from the
// filesystem. Production Docker builds populate it via a separate embed file.
var frontendFS embed.FS

func main() {
	log.Println("Starting FLACidal Server...")

	// Kick off a background refresh of Tidal endpoints from the gist before the downloader initializes.
	core.InitTidalEndpoints()

	// Load config, letting env vars take precedence over the file
	config, err := core.LoadConfigWithEnv()
	if err != nil {
		log.Printf("Warning: could not load config (%v); falling back to defaults", err)
		config = core.GetDefaultConfig()
	}

	downloadDir := config.DownloadFolder
	if downloadDir == "" {
		downloadDir = core.GetDefaultDownloadFolder()
	}
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		log.Printf("Warning: could not create download directory: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := core.NewDatabase()
	if err != nil {
		log.Printf("Warning: could not initialize database: %v", err)
	}

	downloader := core.NewTidalHifiService()

	// Priority (self-hosted) endpoint wiring. The downloader, tidalSource's
	// service, and Qobuz's proxy pool never get a base SetEndpoints/
	// SetProxyEndpoints call in this file (they run on their hardcoded
	// package-default base pools — a separate, pre-existing gap, not fixed
	// here), so there's no SetEndpoints-then-SetPriorityEndpoints ordering
	// risk for those three: SetEndpoints wipes all tiers if it runs after
	// SetPriorityEndpoints, but it's simply never called against any of
	// them. (Qobuz's catalog list IS given a SetEndpoints call below, but
	// that list has no SetPriorityEndpoints counterpart to race against —
	// see the note at that call site.)
	tidalPriority := core.ResolvePriorityEndpoints(config.TidalPriorityEndpoints, config.TidalCustomEndpoint)
	applyPriorityEndpoints("Tidal HiFi (downloader)", downloader.SetPriorityEndpoints, tidalPriority)

	workers := config.ConcurrentDownloads
	if workers <= 0 {
		workers = 4
	}
	downloadManager := core.NewDownloadManager(downloader, workers)

	tidalSource := core.NewTidalSource()
	// TidalSource owns a second, independent TidalHifiService (its own
	// endpoint pool) behind every album/playlist/track fetch — it needs the
	// identical priority list, same as Core's own wiring does.
	applyPriorityEndpoints("Tidal HiFi (source)", tidalSource.GetService().SetPriorityEndpoints, tidalPriority)

	qobuzSource := core.NewQobuzSource(config.QobuzAppID, config.QobuzAppSecret)
	if config.QobuzAuthToken != "" {
		qobuzSource.SetCredentials(config.QobuzAppID, config.QobuzAppSecret, config.QobuzAuthToken)
	}

	qobuzPriority := core.ResolvePriorityEndpoints(config.QobuzPriorityEndpoints, config.QobuzCustomEndpoint)
	applyPriorityEndpoints("Qobuz proxy", qobuzSource.SetProxyPriorityEndpoints, qobuzPriority)
	// The proxy pool above is only half of Qobuz: catalog calls (track/album/
	// playlist/search) go through the separate, tier-less q.endpoints list
	// instead, so priority entries are prepended ahead of the public catalog
	// defaults here too, mirroring Core's own qobuzSource.SetEndpoints call.
	// Deliberately not gated on a QobuzEndpoints override field, since this
	// binary doesn't wire that (unrelated, pre-existing base-endpoint gap);
	// the copy-before-append avoids mutating config.QobuzPriorityEndpoints's
	// backing array through append's slice aliasing.
	if len(qobuzPriority) > 0 {
		catalogEndpoints := append(append([]string{}, qobuzPriority...), core.DefaultQobuzEndpoints()...)
		// No ordering hazard here: q.endpoints (the catalog list) has no
		// SetPriorityEndpoints counterpart, so this SetEndpoints call can't
		// wipe or race against a tier1 set on the same target.
		qobuzSource.SetEndpoints(catalogEndpoints)
		log.Printf("Qobuz catalog endpoints: %d self-hosted configured ahead of the public defaults", len(qobuzPriority))
	}

	// Set up the source manager
	sourceManager := core.NewSourceManager()
	sourceManager.RegisterSource(tidalSource)
	sourceManager.RegisterSource(qobuzSource)
	registerSoulseekSource(sourceManager, config)
	sourceManager.SetPreferredSource(config.PreferredSource)

	lyricsClient := core.NewLyricsClient()

	server := api.NewServer(api.ServerConfig{
		Config:          config,
		DB:              db,
		DownloadManager: downloadManager,
		SourceManager:   sourceManager,
		TidalSource:     tidalSource,
		QobuzSource:     qobuzSource,
		LyricsClient:    lyricsClient,
		Context:         ctx,
		FrontendFS:      frontendFS,
		FrontendDir:     os.Getenv("FRONTEND_DIST_DIR"),
	})

	downloadManager.SetProgressCallback(func(trackID int, status string, result *core.DownloadResult) {
		server.BroadcastDownloadEvent(core.DownloadEvent{
			TrackID: trackID,
			Status:  status,
			Result:  result,
		})
	})

	downloadManager.Start()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down gracefully...")
		cancel()
		downloadManager.Stop()
		if db != nil {
			db.Close()
		}
		_ = server.Shutdown()
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on :%s", port)
	if err := server.Listen(":" + port); err != nil {
		log.Fatalf("Server error: %v", err) //nolint:gocritic // exiting anyway; the deferred cancel() has nothing left to clean up
	}
}

// applyPriorityEndpoints hands a configured self-host list to a source's
// exported SetPriorityEndpoints-style wrapper (setPriority) and logs how many
// survived filtering — the underlying Core call silently drops anything that
// isn't https://, or http:// on a loopback/private address, which from the
// outside is indistinguishable from the setting being ignored altogether.
func applyPriorityEndpoints(label string, setPriority func([]string) int, urls []string) {
	if len(urls) == 0 {
		return
	}
	accepted := setPriority(urls)
	if rejected := len(urls) - accepted; rejected > 0 {
		log.Printf("%s priority endpoints: %d configured, %d rejected (needs https://, or http:// on a loopback/private address)", label, accepted, rejected)
		return
	}
	log.Printf("%s priority endpoints: %d configured (self-hosted, tried first)", label, accepted)
}

// registerSoulseekSource builds a Soulseek source and, when it's enabled and
// reachable, registers it with sm. This mirrors the Soulseek-init step that
// internal/app runs during Startup — this headless binary needs its own copy,
// since otherwise sourceManager would never learn a Soulseek source exists and
// couldn't fall back to it, even after handleSetSourceOrder is fixed.
func registerSoulseekSource(sm *core.SourceManager, config *core.Config) {
	sldlPath := config.SoulseekBinaryPath
	if sldlPath == "" {
		sldlPath = app.DefaultSldlPath()
	}
	if err := app.EnsureSldlExecutable(sldlPath); err != nil {
		log.Printf("Warning: sldl binary might not be executable: %v", err)
	}
	soulseekSource := core.NewSoulseekSource(sldlPath, config.SoulseekUsername, config.SoulseekPassword)
	if config.SoulseekEnabled && soulseekSource.IsAvailable() {
		sm.RegisterSource(soulseekSource)
		log.Println("Registered Soulseek as a fallback source")
	} else if config.SoulseekEnabled {
		log.Println("Warning: Soulseek is enabled but unreachable (check the binary path and credentials)")
	}
}
