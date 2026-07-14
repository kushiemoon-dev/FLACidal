package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// Tests for POST /api/lyrics/file, POST /api/lyrics/fetch-embed and
// POST /api/lyrics/fetch-embed/multiple.
//
// NOT tested here (documented, not fixed): success paths reach a live
// network call to LRCLIB with no injectable HTTP seam (same limitation as
// internal/app's lyrics tests). Only validation and fail-fast "invalid file"
// branches (which return before any network call, since core.ReadFLACMetadata
// errors first) are exercised.

func TestHandleFetchLyricsForFile_MissingFilePath(t *testing.T) {
	s := newTestServer(t)

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/lyrics/file", map[string]interface{}{}, &body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleFetchLyricsForFile_InvalidFile(t *testing.T) {
	s := newTestServer(t)

	path := filepath.Join(t.TempDir(), "not-a-real-flac.flac")
	if err := os.WriteFile(path, []byte("nope"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/lyrics/file", map[string]interface{}{
		"filePath": path,
	}, &body)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusInternalServerError)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleFetchAndEmbedLyrics_MissingFilePath(t *testing.T) {
	s := newTestServer(t)

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/lyrics/fetch-embed", map[string]interface{}{}, &body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleFetchAndEmbedLyrics_InvalidFile(t *testing.T) {
	s := newTestServer(t)

	path := filepath.Join(t.TempDir(), "not-a-real-flac.flac")
	if err := os.WriteFile(path, []byte("nope"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/lyrics/fetch-embed", map[string]interface{}{
		"filePath": path,
	}, &body)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusInternalServerError)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleFetchAndEmbedMultiple_InvalidFiles(t *testing.T) {
	s := newTestServer(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "not-a-real-flac.flac")
	if err := os.WriteFile(path, []byte("nope"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	var results []map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/lyrics/fetch-embed/multiple", map[string]interface{}{
		"filePaths": []string{path},
	}, &results)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if len(results) != 1 {
		t.Fatalf("results = %v, want 1 entry", results)
	}
	if results[0]["success"] != false {
		t.Errorf("results[0] = %v, want success=false", results[0])
	}
	if _, ok := results[0]["error"]; !ok {
		t.Errorf("results[0] = %v, want an 'error' key", results[0])
	}
}

func TestHandleFetchAndEmbedMultiple_EmptyList(t *testing.T) {
	s := newTestServer(t)

	var results []map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/lyrics/fetch-embed/multiple", map[string]interface{}{
		"filePaths": []string{},
	}, &results)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if len(results) != 0 {
		t.Errorf("results = %v, want empty", results)
	}
}
