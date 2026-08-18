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

	workers := config.ConcurrentDownloads
	if workers <= 0 {
		workers = 4
	}
	downloadManager := core.NewDownloadManager(downloader, workers)

	tidalSource := core.NewTidalSource()
	qobuzSource := core.NewQobuzSource(config.QobuzAppID, config.QobuzAppSecret)
	if config.QobuzAuthToken != "" {
		qobuzSource.SetCredentials(config.QobuzAppID, config.QobuzAppSecret, config.QobuzAuthToken)
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
