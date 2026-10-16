package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"flacidal/internal/api"
	"flacidal/internal/app"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// frontendFS stays empty by default; the server falls back to serving from the
// filesystem. Production Docker builds populate it via a separate embed file.
var frontendFS embed.FS

// version is injected at build time via -ldflags "-X main.version=...", read
// from wails.json (the same source of truth the desktop build uses). Plain
// `go build` (e.g. the CI smoke check) leaves it at "dev".
var version = "dev"

// shutdownTimeout bounds how long open connections can delay a shutdown.
const shutdownTimeout = 5 * time.Second

// runDeps carries the one thing run must not do for real under test: the
// background network refresh of the Tidal endpoints.
type runDeps struct{ initEndpoints func() }

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Once shutdown starts, hand the signals back so a second Ctrl-C kills us.
	go func() {
		<-ctx.Done()
		stop()
	}()

	if err := run(ctx, os.Getenv, runDeps{initEndpoints: core.InitTidalEndpoints}); err != nil {
		log.Fatalf("%v", err)
	}
}

// run wires and serves the headless API until ctx is cancelled or the listener
// fails. Everything it opens is closed before it returns.
func run(ctx context.Context, getenv func(string) string, deps runDeps) error {
	log.Println("Starting FLACidal Server...")

	listen, err := resolveListenSettings(getenv)
	if err != nil {
		return err
	}
	if listen.Token == "" {
		log.Println("WARNING: no AUTH_TOKEN set, authentication is disabled (loopback only); any local process can use the API")
	} else {
		log.Println("Authentication enabled")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Core builds config, database, sources, download manager and orchestrator;
	// the environment variables win over the config file at load time only.
	c, err := core.New("", core.Options{LoadConfig: core.LoadConfigWithEnv, InitEndpoints: deps.initEndpoints})
	if err != nil {
		return fmt.Errorf("could not start the core: %w", err)
	}
	config := c.Config()

	downloadDir := config.DownloadFolder
	if downloadDir == "" {
		downloadDir = core.GetDefaultDownloadFolder()
	}
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		log.Printf("Warning: could not create download directory: %v", err)
	}

	lyricsClient := core.NewLyricsClient()

	// The sldl handle only backs the Soulseek status and test screens; downloads use Core's native client.
	soulseekSource := app.NewSldlSource(&config, c.LogBuffer())
	log.Println("Soulseek downloads use the native client; the sldl binary is only used for the status screens")

	server := api.NewServer(api.ServerConfig{
		Config:          &config,
		Core:            c,
		LogBuffer:       c.LogBuffer(),
		DB:              c.DB(),
		DownloadManager: c.DownloadManager(),
		SourceManager:   c.SourceManager(),
		TidalSource:     c.TidalSource(),
		QobuzSource:     c.QobuzSource(),
		LyricsClient:    lyricsClient,
		Downloader:      c.Downloader(),
		SoulseekSource:  soulseekSource,
		Version:         version,
		Context:         ctx,
		FrontendFS:      frontendFS,
		FrontendDir:     getenv("FRONTEND_DIST_DIR"),

		AuthToken:        listen.Token,
		EnforceLocalHost: listen.Loopback && listen.Token == "",
		CORSOrigins:      listen.CORSOrigins,
	})

	c.DownloadManager().SetProgressCallback(func(trackID int, status string, result *core.DownloadResult) {
		server.BroadcastDownloadEvent(core.DownloadEvent{
			TrackID: trackID,
			Status:  status,
			Result:  result,
		})
	})

	server.StartHealthProbe()

	// Starts the workers and restores the persisted queue; the callback above is already in place.
	c.Start()

	shutdown := func(ln net.Listener) {
		shutdownAll(c.DownloadManager().Stop, c.Close, func() error { return server.ShutdownWithTimeout(shutdownTimeout) }, ln, shutdownTimeout)
	}

	ln, err := net.Listen("tcp", listen.Addr)
	if err != nil {
		cancel()
		shutdown(nil)
		return fmt.Errorf("server error: %w", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		<-ctx.Done()

		log.Println("Shutting down gracefully...")
		shutdown(ln)
	}()

	log.Printf("Listening on %s", listen.Addr)
	listenErr := server.Serve(ln)
	cancel()
	<-done
	if listenErr != nil {
		return fmt.Errorf("server error: %w", listenErr)
	}
	return nil
}

// shutdownAll stops the HTTP server first (bounded by timeout), closes the
// listener (when there is one) so Serve returns even if the shutdown raced its start, then waits at
// most timeout for the download workers. The database is only closed once the
// workers are really done, since one still running could write to it.
func shutdownAll(stopDownloads, closeDB func(), shutdownHTTP func() error, ln net.Listener, timeout time.Duration) {
	if err := shutdownHTTP(); err != nil {
		log.Printf("Warning: shutdown did not finish cleanly: %v", err)
	}
	if ln != nil {
		_ = ln.Close()
	}

	stopped := make(chan struct{})
	go func() {
		stopDownloads()
		close(stopped)
	}()
	select {
	case <-stopped:
		closeDB()
	case <-time.After(timeout):
		log.Printf("Warning: downloads did not stop within %s, leaving the database open", timeout)
	}
}
