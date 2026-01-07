package api

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"

	"flacidal/backend"
)

// ServerConfig holds all dependencies for the server
type ServerConfig struct {
	Config          *backend.Config
	DB              *backend.Database
	DownloadManager *backend.DownloadManager
	SourceManager   *backend.SourceManager
	TidalSource     *backend.TidalSource
	QobuzSource     *backend.QobuzSource
	LyricsClient    *backend.LyricsClient
	Context         context.Context
	FrontendFS      embed.FS // Embedded frontend assets
}

// Server represents the HTTP API server
type Server struct {
	app             *fiber.App
	config          *backend.Config
	db              *backend.Database
	downloadManager *backend.DownloadManager
	sourceManager   *backend.SourceManager
	tidalSource     *backend.TidalSource
	qobuzSource     *backend.QobuzSource
	lyricsClient    *backend.LyricsClient
	wsHub           *WebSocketHub
	ctx             context.Context
	frontendFS      embed.FS
}

// NewServer creates a new API server instance
func NewServer(cfg ServerConfig) *Server {
	app := fiber.New(fiber.Config{
		AppName:      "FLACidal Server",
		ServerHeader: "FLACidal",
		BodyLimit:    50 * 1024 * 1024, // 50MB
	})

	// Create WebSocket hub
	wsHub := NewWebSocketHub()
	go wsHub.Run()

	server := &Server{
		app:             app,
		config:          cfg.Config,
		db:              cfg.DB,
		downloadManager: cfg.DownloadManager,
		sourceManager:   cfg.SourceManager,
		tidalSource:     cfg.TidalSource,
		qobuzSource:     cfg.QobuzSource,
		lyricsClient:    cfg.LyricsClient,
		wsHub:           wsHub,
		ctx:             cfg.Context,
		frontendFS:      cfg.FrontendFS,
	}

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${latency})\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Setup routes
	server.setupRoutes()

	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check
	s.app.Get("/api/health", s.handleHealth)

	// API routes
	api := s.app.Group("/api")

	// Config routes
	api.Get("/config", s.handleGetConfig)
	api.Post("/config", s.handleSaveConfig)
	api.Post("/config/reset", s.handleResetConfig)

	// Source routes
	api.Get("/sources", s.handleGetSources)
	api.Get("/sources/preferred", s.handleGetPreferredSource)
	api.Post("/sources/preferred", s.handleSetPreferredSource)
	api.Post("/sources/detect", s.handleDetectSource)

	// Content routes (Tidal/Qobuz)
	api.Post("/content/fetch", s.handleFetchContent)
	api.Post("/content/validate", s.handleValidateURL)
	api.Get("/content/search", s.handleSearch)

	// Download routes
	api.Get("/downloads/queue", s.handleGetQueue)
	api.Post("/downloads/queue", s.handleQueueDownloads)
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

	// History routes
	api.Get("/history", s.handleGetHistory)
	api.Get("/history/filtered", s.handleGetHistoryFiltered)
	api.Delete("/history/:id", s.handleDeleteHistory)
	api.Post("/history/clear", s.handleClearHistory)
	api.Post("/history/refetch/:id", s.handleRefetchFromHistory)

	// Files routes
	api.Get("/files", s.handleListFiles)
	api.Delete("/files", s.handleDeleteFile)
	api.Get("/files/metadata", s.handleGetMetadata)
	api.Get("/files/cover", s.handleGetCoverArt)
	api.Get("/files/templates", s.handleGetRenameTemplates)
	api.Post("/files/rename/preview", s.handlePreviewRename)
	api.Post("/files/rename", s.handleRenameFiles)

	// Conversion routes
	api.Get("/convert/available", s.handleIsConverterAvailable)
	api.Get("/convert/ffmpeg", s.handleGetFFmpegInfo)
	api.Get("/convert/formats", s.handleGetConversionFormats)
	api.Post("/convert", s.handleConvertFiles)

	// Analysis routes
	api.Post("/analyze", s.handleAnalyzeFile)
	api.Post("/analyze/multiple", s.handleAnalyzeMultiple)
	api.Post("/analyze/quick", s.handleQuickAnalyze)

	// Lyrics routes
	api.Get("/lyrics", s.handleFetchLyrics)
	api.Post("/lyrics/file", s.handleFetchLyricsForFile)
	api.Post("/lyrics/embed", s.handleEmbedLyrics)
	api.Post("/lyrics/fetch-embed", s.handleFetchAndEmbedLyrics)
	api.Post("/lyrics/fetch-embed/multiple", s.handleFetchAndEmbedMultiple)

	// Qobuz routes
	api.Post("/qobuz/credentials", s.handleUpdateQobuzCredentials)
	api.Get("/qobuz/configured", s.handleIsQobuzConfigured)

	// Folder routes
	api.Get("/folder", s.handleGetDownloadFolder)
	api.Post("/folder", s.handleSetDownloadFolder)

	// System routes
	api.Get("/version", s.handleGetVersion)
	api.Get("/logs", s.handleGetLogs)
	api.Post("/logs/clear", s.handleClearLogs)
	api.Get("/connection", s.handleGetConnectionStatus)
	api.Get("/downloader/available", s.handleIsDownloaderAvailable)

	// WebSocket endpoint
	s.app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	s.app.Get("/ws", websocket.New(s.handleWebSocket))

	// Static files (Svelte build) - serve embedded frontend
	frontendDist, err := fs.Sub(s.frontendFS, "frontend/dist")
	if err == nil {
		s.app.Use("/", filesystem.New(filesystem.Config{
			Root:         http.FS(frontendDist),
			Browse:       false,
			Index:        "index.html",
			NotFoundFile: "index.html", // SPA fallback
		}))
	} else {
		// Fallback to file system for development
		s.app.Static("/", "./frontend/dist")
		s.app.Get("/*", func(c *fiber.Ctx) error {
			return c.SendFile("./frontend/dist/index.html")
		})
	}
}

// Listen starts the HTTP server
func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	s.wsHub.Close()
	return s.app.Shutdown()
}

// BroadcastDownloadEvent sends a download event to all connected WebSocket clients
func (s *Server) BroadcastDownloadEvent(event backend.DownloadEvent) {
	s.wsHub.Broadcast(map[string]interface{}{
		"type":    "download-progress",
		"trackId": event.TrackID,
		"status":  event.Status,
		"result":  event.Result,
	})
}

// WebSocketHub manages WebSocket connections
type WebSocketHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan interface{}
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
	done       chan struct{}
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan interface{}, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		done:       make(chan struct{}),
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	for {
		select {
		case <-h.done:
			return
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected (total: %d)", len(h.clients))
		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected (total: %d)", len(h.clients))
		case message := <-h.broadcast:
			h.mu.RLock()
			for conn := range h.clients {
				if err := conn.WriteJSON(message); err != nil {
					log.Printf("WebSocket write error: %v", err)
					h.mu.RUnlock()
					h.unregister <- conn
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *WebSocketHub) Broadcast(message interface{}) {
	select {
	case h.broadcast <- message:
	default:
		log.Println("WebSocket broadcast channel full, dropping message")
	}
}

// Close shuts down the hub
func (h *WebSocketHub) Close() {
	close(h.done)
	h.mu.Lock()
	for conn := range h.clients {
		conn.Close()
	}
	h.mu.Unlock()
}

// handleWebSocket handles WebSocket connections
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
