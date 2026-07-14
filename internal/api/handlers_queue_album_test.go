package api

import (
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Tests for POST /api/downloads/queue/album.

func TestHandleQueueArtistAlbum_MissingOutputDir(t *testing.T) {
	s := newTestServer(t)

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/downloads/queue/album", map[string]interface{}{
		"albumId": "123",
	}, &body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleQueueArtistAlbum_NoDownloadManager(t *testing.T) {
	s := NewServer(ServerConfig{Config: &core.Config{}})

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/downloads/queue/album", map[string]interface{}{
		"albumId":   "123",
		"outputDir": "/tmp",
	}, &body)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusInternalServerError)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}
