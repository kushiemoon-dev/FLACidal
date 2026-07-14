package api

import (
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Tests for GET /api/content/search/albums, /artists and /deezer.
//
// NOT tested here (documented, not fixed): the success path makes a live
// network call (Tidal proxy / Deezer public API) with no injectable HTTP
// seam, matching the existing limitation documented in handlers_search_test.go.
// Only validation and nil-dependency branches are exercised.

func TestHandleSearchTidalAlbums_MissingQuery(t *testing.T) {
	s := newTestServer(t)

	var body map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/content/search/albums", nil, &body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleSearchTidalAlbums_NoTidalSource(t *testing.T) {
	s := NewServer(ServerConfig{Config: &core.Config{}})

	var body map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/content/search/albums?q=daft+punk", nil, &body)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusInternalServerError)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleSearchTidalArtists_MissingQuery(t *testing.T) {
	s := newTestServer(t)

	var body map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/content/search/artists", nil, &body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleSearchTidalArtists_NoTidalSource(t *testing.T) {
	s := NewServer(ServerConfig{Config: &core.Config{}})

	var body map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/content/search/artists?q=daft+punk", nil, &body)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusInternalServerError)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleSearchDeezer_EmptyQuery(t *testing.T) {
	s := newTestServer(t)

	var results []map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/content/search/deezer", nil, &results)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if len(results) != 0 {
		t.Errorf("results = %v, want empty for an empty query", results)
	}
}
