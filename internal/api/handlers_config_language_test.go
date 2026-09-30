package api

import (
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func TestHandleSaveConfig_RoundTripsLanguage(t *testing.T) {
	core.SetDataDir(t.TempDir())
	s := NewServer(ServerConfig{Config: &core.Config{}})

	resp := doRequest(t, s, "POST", "/api/config", map[string]interface{}{"language": "fr"}, nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("save status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}

	var got core.Config
	doRequest(t, s, "GET", "/api/config", nil, &got)
	if got.Language != "fr" {
		t.Errorf("GET language = %q, want fr", got.Language)
	}

	loaded, err := core.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loaded.Language != "fr" {
		t.Errorf("persisted language = %q, want fr", loaded.Language)
	}
}
