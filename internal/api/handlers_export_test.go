package api

import (
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Tests for GET /api/downloads/export.

func TestHandleExportFailedDownloads_NoDownloadManager(t *testing.T) {
	s := newTestServer(t)

	var body map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/downloads/export", nil, &body)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusInternalServerError)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleExportFailedDownloads_NoFailedJobs(t *testing.T) {
	s := NewServer(ServerConfig{
		Config:          &core.Config{},
		DownloadManager: core.NewDownloadManager(core.NewTidalHifiService(), 1),
	})

	var body map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/downloads/export", nil, &body)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusNotFound)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}
