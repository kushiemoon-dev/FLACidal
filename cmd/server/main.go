package main

import (
	"context"
	"embed"
	"log"
	"os"
	"os/signal"
	"syscall"

	core "github.com/kushiemoon-dev/flacidal-core"
	"flacidal/internal/api"
)

// frontendFS is empty by default - the server will serve from filesystem
// For production Docker builds, this is populated by a separate embed file
var frontendFS embed.FS

func main() {
	log.Println("FLACidal Server starting...")

	// Load config (env vars override file config)
	config, err := core.LoadConfigWithEnv()
	if err != nil {
		log.Printf("Warning: Could not load config: %v, using defaults", err)
		config = core.GetDefaultConfig()
	}

	// Ensure download directory exists
	downloadDir := config.DownloadFolder
	if downloadDir == "" {
		downloadDir = core.GetDefaultDownloadFolder()
	}
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		log.Printf("Warning: Could not create download directory: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database
	db, err := core.NewDatabase()
	if err != nil {
		log.Printf("Warning: Could not initialize database: %v", err)
	}

	// Initialize FLAC downloader service
	downloader := core.NewTidalHifiService()

	// Initialize download manager
	workers := config.ConcurrentDownloads
	if workers <= 0 {
		workers = 4
	}
	downloadManager := core.NewDownloadManager(downloader, workers)

	// Initialize sources
	tidalSource := core.NewTidalSource()
	qobuzSource := core.NewQobuzSource(config.QobuzAppID, config.QobuzAppSecret)
	if config.QobuzAuthToken != "" {
		qobuzSource.SetCredentials(config.QobuzAppID, config.QobuzAppSecret, config.QobuzAuthToken)
	}

	// Initialize source manager
	sourceManager := core.NewSourceManager()
	sourceManager.RegisterSource(tidalSource)
	sourceManager.RegisterSource(qobuzSource)
	sourceManager.SetPreferredSource(config.PreferredSource)

	// Initialize lyrics client
	lyricsClient := core.NewLyricsClient()

	// Create and configure server
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
	})

	// Set download progress callback to broadcast via WebSocket
	downloadManager.SetProgressCallback(func(trackID int, status string, result *core.DownloadResult) {
		server.BroadcastDownloadEvent(core.DownloadEvent{
			TrackID: trackID,
			Status:  status,
			Result:  result,
		})
	})

	// Start download manager
	downloadManager.Start()

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down...")
		cancel()
		downloadManager.Stop()
		if db != nil {
			db.Close()
		}
		server.Shutdown()
	}()

	// Get port from env or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on :%s", port)
	if err := server.Listen(":" + port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
