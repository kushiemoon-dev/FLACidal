package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Tests for GET /api/files, GET /api/files/metadata and GET /api/files/cover.
// Mirrors internal/app's characterization-test style: exercise fail-fast
// branches (no config, invalid FLAC content) rather than real audio decoding.

func TestHandleListFiles_NoDownloadFolder(t *testing.T) {
	s := newTestServer(t)

	var files []core.DownloadedFileInfo
	resp := doRequest(t, s, "GET", "/api/files", nil, &files)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if len(files) != 0 {
		t.Errorf("files = %v, want empty (no download folder configured)", files)
	}
}

func TestHandleListFiles_EmptyDownloadFolder(t *testing.T) {
	dir := t.TempDir()
	s := NewServer(ServerConfig{
		Config: &core.Config{DownloadFolder: dir},
	})

	var files []core.DownloadedFileInfo
	resp := doRequest(t, s, "GET", "/api/files", nil, &files)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if len(files) != 0 {
		t.Errorf("files = %v, want empty (empty folder)", files)
	}
}

func TestHandleGetMetadata_MissingPath(t *testing.T) {
	s := newTestServer(t)

	var body map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/files/metadata", nil, &body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleGetMetadata_InvalidFile(t *testing.T) {
	s := newTestServer(t)

	path := filepath.Join(t.TempDir(), "not-a-real-flac.flac")
	if err := os.WriteFile(path, []byte("not actually flac data"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	var body map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/files/metadata?path="+path, nil, &body)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusInternalServerError)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleGetCoverArt_MissingPath(t *testing.T) {
	s := newTestServer(t)

	var body map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/files/cover", nil, &body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleGetCoverArt_InvalidFile(t *testing.T) {
	s := newTestServer(t)

	path := filepath.Join(t.TempDir(), "not-a-real-flac.flac")
	if err := os.WriteFile(path, []byte("not actually flac data"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	var body map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/files/cover?path="+path, nil, &body)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusInternalServerError)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}
