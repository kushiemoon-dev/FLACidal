package api

import (
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Tests for POST /api/downloads/queue/qobuz.

func TestHandleQueueQobuzDownloads_NoDownloadManager(t *testing.T) {
	s := newTestServer(t)

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/downloads/queue/qobuz", map[string]interface{}{
		"tracks":    []core.SourceTrack{},
		"outputDir": "/tmp",
	}, &body)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusInternalServerError)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleQueueQobuzDownloads_MissingOutputDir(t *testing.T) {
	s := NewServer(ServerConfig{
		Config:          &core.Config{},
		DownloadManager: core.NewDownloadManager(core.NewTidalHifiService(), 1),
	})

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/downloads/queue/qobuz", map[string]interface{}{
		"tracks": []core.SourceTrack{},
	}, &body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleQueueQobuzDownloads_EmptyTracks(t *testing.T) {
	s := NewServer(ServerConfig{
		Config:          &core.Config{},
		DownloadManager: core.NewDownloadManager(core.NewTidalHifiService(), 1),
	})

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/downloads/queue/qobuz", map[string]interface{}{
		"tracks":    []core.SourceTrack{},
		"outputDir": "/tmp",
	}, &body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if body["queued"] != float64(0) {
		t.Errorf("queued = %v, want 0", body["queued"])
	}
}
