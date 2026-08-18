package api

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// defaultFrontendDir is where the compiled Svelte SPA lives on disk; it's
// used only when frontendFS doesn't carry an embedded build.
const defaultFrontendDir = "frontend/dist"

type ServerConfig struct {
	Config          *core.Config
	DB              *core.Database
	DownloadManager *core.DownloadManager
	SourceManager   *core.SourceManager
	TidalSource     *core.TidalSource
	QobuzSource     *core.QobuzSource
	LyricsClient    *core.LyricsClient
	Context         context.Context
	FrontendFS      embed.FS
	FrontendDir     string // on-disk SPA path used when FrontendFS is empty (default: "frontend/dist")
}

type Server struct {
	app              *fiber.App
	config           *core.Config
	db               *core.Database
	downloadManager  *core.DownloadManager
	sourceManager    *core.SourceManager
	tidalSource      *core.TidalSource
	qobuzSource      *core.QobuzSource
	lyricsClient     *core.LyricsClient
	wsHub            *WebSocketHub
	queueBroadcaster *QueueBroadcaster
	ctx              context.Context
	frontendFS       embed.FS
	frontendDir      string
}

func NewServer(cfg ServerConfig) *Server {
	app := fiber.New(fiber.Config{
		AppName:      "FLACidal Server",
		ServerHeader: "FLACidal",
		BodyLimit:    50 * 1024 * 1024,
	})

	wsHub := NewWebSocketHub()
	go wsHub.Run()

	queueBroadcaster := NewQueueBroadcaster()

	frontendDir := cfg.FrontendDir
	if frontendDir == "" {
		frontendDir = defaultFrontendDir
	}

	server := &Server{
		app:              app,
		config:           cfg.Config,
		db:               cfg.DB,
		downloadManager:  cfg.DownloadManager,
		sourceManager:    cfg.SourceManager,
		tidalSource:      cfg.TidalSource,
		qobuzSource:      cfg.QobuzSource,
		lyricsClient:     cfg.LyricsClient,
		wsHub:            wsHub,
		queueBroadcaster: queueBroadcaster,
		ctx:              cfg.Context,
		frontendFS:       cfg.FrontendFS,
		frontendDir:      frontendDir,
	}

	// So every queued/downloading/completed/failed transition reaches WS subscribers.
	if cfg.DownloadManager != nil {
		cfg.DownloadManager.SetProgressCallback(func(trackID int, status string, result *core.DownloadResult) {
			jobID := fmt.Sprintf("%d", trackID)
			event := QueueEvent{JobID: jobID}

			title := ""
			artist := ""
			if result != nil {
				if result.Title != "" {
					title = result.Title
				}
				if result.Artist != "" {
					artist = result.Artist
				}
			}
			event.Title = title
			event.Artist = artist

			switch status {
			case "queued":
				event.Type = "queued"
			case "downloading":
				event.Type = "started"
				if result != nil && result.BytesTotal > 0 {
					event.Type = "progress"
					event.Progress = int(result.BytesDownloaded * 100 / result.BytesTotal)
				}
			case "completed":
				event.Type = "completed"
			case "error", "cancelled":
				event.Type = "failed"
				if result != nil {
					event.Error = result.Error
				}
			default:
				return
			}

			queueBroadcaster.Broadcast(event)
		})

		cfg.DownloadManager.SetJobCompleteCallback(func(entry core.HistoryEntry) {
			if cfg.DB != nil {
				if err := cfg.DB.InsertHistoryEntry(entry); err != nil {
					log.Printf("WARN: could not save history entry: %v", err)
				}
			}
		})
	}

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${latency})\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	server.setupRoutes()

	return server
}

func (s *Server) setupRoutes() {
	s.app.Get("/api/health", s.handleHealth)

	api := s.app.Group("/api")

	api.Get("/config", s.handleGetConfig)
	api.Post("/config", s.handleSaveConfig)
	api.Post("/config/reset", s.handleResetConfig)

	api.Get("/sources", s.handleGetSources)
	api.Get("/sources/preferred", s.handleGetPreferredSource)
	api.Post("/sources/preferred", s.handleSetPreferredSource)
	api.Post("/sources/detect", s.handleDetectSource)
	api.Post("/sources/order", s.handleSetSourceOrder)
	api.Get("/sources/soulseek/status", s.handleGetSldlStatus)
	api.Post("/sources/soulseek/test", s.handleTestSoulseekConnection)

	api.Post("/content/fetch", s.handleFetchContent)
	api.Post("/content/validate", s.handleValidateURL)
	api.Get("/content/search", s.handleSearch)
	api.Get("/content/search/albums", s.handleSearchTidalAlbums)
	api.Get("/content/search/artists", s.handleSearchTidalArtists)
	api.Get("/content/search/deezer", s.handleSearchDeezer)

	api.Get("/downloads/queue", s.handleGetQueue)
	api.Post("/downloads/queue", s.handleQueueDownloads)
	api.Post("/downloads/queue/album", s.handleQueueArtistAlbum)
	api.Post("/downloads/queue/qobuz", s.handleQueueQobuzDownloads)
	api.Post("/downloads/single", s.handleQueueSingle)
	api.Get("/downloads/status", s.handleGetQueueStatus)
	api.Get("/downloads/options", s.handleGetDownloadOptions)
	api.Post("/downloads/options", s.handleSetDownloadOptions)
	api.Post("/downloads/retry/:id", s.handleRetryDownload)
	api.Post("/downloads/retry-all", s.handleRetryAllFailed)
	api.Post("/downloads/cancel/:id", s.handleCancelDownload)
	api.Post("/downloads/pause", s.handlePauseDownloads)
	api.Post("/downloads/resume", s.handleResumeDownloads)
	api.Get("/downloads/paused", s.handleIsPaused)
	api.Get("/downloads/export", s.handleExportFailedDownloads)

	api.Get("/history", s.handleGetHistory)
	api.Get("/history/filtered", s.handleGetHistoryFiltered)
	api.Delete("/history/:id", s.handleDeleteHistory)
	api.Post("/history/clear", s.handleClearHistory)
	api.Post("/history/refetch/:id", s.handleRefetchFromHistory)
	api.Get("/history/recent", s.handleGetRecentAlbums)

	api.Get("/files", s.handleListFiles)
	api.Delete("/files", s.handleDeleteFile)
	api.Get("/files/metadata", s.handleGetMetadata)
	api.Get("/files/cover", s.handleGetCoverArt)
	api.Get("/files/templates", s.handleGetRenameTemplates)
	api.Post("/files/rename/preview", s.handlePreviewRename)
	api.Post("/files/rename", s.handleRenameFiles)

	api.Get("/convert/available", s.handleIsConverterAvailable)
	api.Get("/convert/ffmpeg", s.handleGetFFmpegInfo)
	api.Get("/convert/formats", s.handleGetConversionFormats)
	api.Post("/convert", s.handleConvertFiles)

	RegisterAnalyzerRoutes(api, s)

	api.Get("/lyrics", s.handleFetchLyrics)
	api.Post("/lyrics/file", s.handleFetchLyricsForFile)
	api.Post("/lyrics/embed", s.handleEmbedLyrics)
	api.Post("/lyrics/fetch-embed", s.handleFetchAndEmbedLyrics)
	api.Post("/lyrics/fetch-embed/multiple", s.handleFetchAndEmbedMultiple)

	api.Post("/qobuz/credentials", s.handleUpdateQobuzCredentials)
	api.Get("/qobuz/configured", s.handleIsQobuzConfigured)

	api.Get("/folder", s.handleGetDownloadFolder)
	api.Post("/folder", s.handleSetDownloadFolder)

	api.Get("/version", s.handleGetVersion)
	api.Get("/logs", s.handleGetLogs)
	api.Post("/logs/clear", s.handleClearLogs)
	api.Get("/connection", s.handleGetConnectionStatus)
	api.Get("/downloader/available", s.handleIsDownloaderAvailable)

	RegisterHistoryRoutes(api, s)

	RegisterQueueRoutes(s.app, s)

	s.app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	s.app.Get("/ws", websocket.New(s.handleWebSocket))

	// An embedded build (as used for Docker/production images) takes priority
	// when present. fs.Sub won't error on a syntactically valid path even
	// against an empty embed.FS, so we still have to probe with fs.Stat to
	// know whether anything was actually embedded.
	_, embedErr := fs.Stat(s.frontendFS, "frontend/dist/index.html")
	if embedErr == nil {
		frontendDist, _ := fs.Sub(s.frontendFS, "frontend/dist")
		s.app.Use("/", filesystem.New(filesystem.Config{
			Root:         http.FS(frontendDist),
			Browse:       false,
			Index:        "index.html",
			NotFoundFile: "index.html", // fall through to the SPA for unknown routes
		}))
	} else if indexPath := filepath.Join(s.frontendDir, "index.html"); fileExists(indexPath) {
		// Nothing embedded (typical of `go run ./cmd/server`) - fall back to
		// whatever was built on disk via `cd frontend && npm run build`.
		s.app.Static("/", s.frontendDir)
		s.app.Get("/*", func(c *fiber.Ctx) error {
			return c.SendFile(indexPath)
		})
	} else {
		msg := fmt.Sprintf(
			"No frontend build found. Run `cd frontend && npm install && npm run build` "+
				"(or `make serve`) and restart the server. Checked path: %s", indexPath)
		log.Printf("WARN: %s", msg)
		s.app.Get("/*", func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusServiceUnavailable).SendString(msg)
		})
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}

// Shutdown closes WS connections first, before shutting down the app.
func (s *Server) Shutdown() error {
	s.wsHub.Close()
	return s.app.Shutdown()
}

func (s *Server) BroadcastDownloadEvent(event core.DownloadEvent) {
	s.wsHub.Broadcast(map[string]interface{}{
		"type":    "download-progress",
		"trackId": event.TrackID,
		"status":  event.Status,
		"result":  event.Result,
	})
}

type WebSocketHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan interface{}
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
	done       chan struct{}
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan interface{}, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		done:       make(chan struct{}),
	}
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case <-h.done:
			return
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()
			log.Printf("WebSocket client joined (now %d connected)", len(h.clients))
		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
			h.mu.Unlock()
			log.Printf("WebSocket client left (now %d connected)", len(h.clients))
		case message := <-h.broadcast:
			h.mu.RLock()
			for conn := range h.clients {
				if err := conn.WriteJSON(message); err != nil {
					log.Printf("WebSocket send failed: %v", err)
					h.mu.RUnlock()
					h.unregister <- conn
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *WebSocketHub) Broadcast(message interface{}) {
	select {
	case h.broadcast <- message:
	default:
		log.Println("WebSocket broadcast queue is full; dropping message")
	}
}

func (h *WebSocketHub) Close() {
	close(h.done)
	h.mu.Lock()
	for conn := range h.clients {
		conn.Close()
	}
	h.mu.Unlock()
}

func (s *Server) handleWebSocket(c *websocket.Conn) {
	s.wsHub.register <- c
	defer func() {
		s.wsHub.unregister <- c
	}()

	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			break
		}
	}
}
