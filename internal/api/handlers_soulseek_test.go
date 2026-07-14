package api

import (
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Tests for GET /api/sources/soulseek/status and POST /api/sources/soulseek/test.
//
// NOT tested here (documented, not fixed): both handlers shell out to a
// local sldl binary that isn't present in CI, so only the "not found" /
// "not installed" branches are exercised — same limitation as internal/app's
// own TestGetSldlStatus / TestTestSoulseekConnection characterization tests.

func TestHandleGetSldlStatus_NotInstalled(t *testing.T) {
	s := NewServer(ServerConfig{
		Config: &core.Config{SoulseekBinaryPath: "/nonexistent/sldl"},
	})

	var body map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/sources/soulseek/status", nil, &body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if body["installed"] != false {
		t.Errorf("installed = %v, want false", body["installed"])
	}
	if body["path"] != "/nonexistent/sldl" {
		t.Errorf("path = %v, want the configured SoulseekBinaryPath", body["path"])
	}
}

func TestHandleTestSoulseekConnection_NotFound(t *testing.T) {
	s := NewServer(ServerConfig{
		Config: &core.Config{SoulseekBinaryPath: "/nonexistent/sldl"},
	})

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/sources/soulseek/test", map[string]interface{}{
		"username": "user",
		"password": "pass",
	}, &body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if body["success"] != false {
		t.Errorf("success = %v, want false", body["success"])
	}
	if body["message"] != "sldl not found" {
		t.Errorf("message = %v, want %q", body["message"], "sldl not found")
	}
}
