package api

import (
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Intentionally out of scope: the success path makes a real network call to
// the Tidal HiFi proxy, with no seam available to substitute a fake HTTP
// client (the same limitation internal/app's SearchTidal tests run into).
// Only the validation and nil-dependency branches are exercised here.

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
