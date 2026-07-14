package api

import (
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Tests for POST /api/sources/order.

func TestHandleSetSourceOrder_Empty(t *testing.T) {
	s := newTestServer(t)

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/sources/order", map[string]interface{}{
		"order": []string{},
	}, &body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleSetSourceOrder_UnknownSource(t *testing.T) {
	s := newTestServer(t)

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/sources/order", map[string]interface{}{
		"order": []string{"tidal", "spotify"},
	}, &body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleSetSourceOrder_Duplicate(t *testing.T) {
	s := newTestServer(t)

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/sources/order", map[string]interface{}{
		"order": []string{"tidal", "tidal"},
	}, &body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleSetSourceOrder_Success(t *testing.T) {
	// core.SaveConfig writes to core.GetDataDir(), a package-level global —
	// redirect it to a temp dir so this never touches a real ~/.flacidal/config.json.
	core.SetDataDir(t.TempDir())
	s := NewServer(ServerConfig{
		Config:          &core.Config{},
		DownloadManager: core.NewDownloadManager(core.NewTidalHifiService(), 1),
	})

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/sources/order", map[string]interface{}{
		"order": []string{"qobuz", "tidal"},
	}, &body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if body["success"] != true {
		t.Errorf("body = %v, want success:true", body)
	}
	if len(s.config.SourceOrder) != 2 || s.config.SourceOrder[0] != "qobuz" {
		t.Errorf("config.SourceOrder = %v, want [qobuz tidal]", s.config.SourceOrder)
	}
}
