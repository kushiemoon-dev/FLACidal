package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Follows the same characterization-test style used in internal/app: it
// drives the fail-fast branches (missing config, bogus FLAC content) instead
// of decoding real audio.

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

	dir := t.TempDir()
	s.config.DownloadFolder = dir
	path := filepath.Join(dir, "not-a-real-flac.flac")
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

	dir := t.TempDir()
	s.config.DownloadFolder = dir
	path := filepath.Join(dir, "not-a-real-flac.flac")
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

func TestHandleFiles_M4aListedAndDeletable(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"song.m4a", "song.mp3"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s := NewServer(ServerConfig{Config: &core.Config{DownloadFolder: dir}})

	var files []core.DownloadedFileInfo
	if resp := doRequest(t, s, "GET", "/api/files", nil, &files); resp.StatusCode != fiber.StatusOK {
		t.Fatalf("list status = %d", resp.StatusCode)
	}
	if len(files) != 1 || files[0].Name != "song.m4a" || files[0].Format != "m4a" {
		t.Fatalf("files = %+v, want only song.m4a", files)
	}

	var body map[string]interface{}
	resp := doRequest(t, s, "DELETE", "/api/files?path="+filepath.Join(dir, "song.mp3"), nil, &body)
	if resp.StatusCode != fiber.StatusBadRequest || body["error"] != "only downloaded FLAC and M4A files can be deleted" {
		t.Errorf("mp3 delete: status=%d body=%v", resp.StatusCode, body)
	}
	resp = doRequest(t, s, "DELETE", "/api/files?path="+filepath.Join(dir, "song.m4a"), nil, nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("m4a delete status = %d", resp.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(dir, "song.m4a")); err == nil {
		t.Error("song.m4a should be gone")
	}
}

func TestHandleDeleteFile_M4aOutsideAllowedFolders(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "x.m4a")
	os.WriteFile(outside, []byte("x"), 0o644)
	s := NewServer(ServerConfig{Config: &core.Config{DownloadFolder: dir}})
	resp := doRequest(t, s, "DELETE", "/api/files?path="+outside, nil, nil)
	if resp.StatusCode == fiber.StatusOK {
		t.Error("outside file must not be deletable")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Error("outside file was deleted")
	}
}
