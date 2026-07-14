package api

import (
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Tests for GET /api/content/search.
//
// NOT tested here (documented, not fixed): the success path makes a live
// network call to the Tidal HiFi proxy with no injectable HTTP seam (same
// limitation as internal/app's SearchTidal tests). Only validation and
// nil-dependency branches are exercised.

func TestHandleSearch_MissingQuery(t *testing.T) {
	s := newTestServer(t)

	var body map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/content/search", nil, &body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleSearch_NoTidalSource(t *testing.T) {
	s := NewServer(ServerConfig{
		Config: &core.Config{},
	})

	var body map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/content/search?q=daft+punk", nil, &body)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusInternalServerError)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}
