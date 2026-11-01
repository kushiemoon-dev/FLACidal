package api

import (
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// The success path of the connection test needs a Soulseek server, so it is
// covered in FLACidal-Core (Core.TestSoulseekLogin against a local listener).
// Here only the branches that never reach the network are exercised.

func soulseekCore(t *testing.T, enabled bool) *core.Core {
	t.Helper()
	dir := t.TempDir()
	core.SetDataDir(dir)
	t.Cleanup(func() { core.SetDataDir("") })
	if err := core.SaveConfig(&core.Config{SoulseekEnabled: enabled}); err != nil {
		t.Fatal(err)
	}
	c, err := core.New(dir, core.Options{InitEndpoints: func() {}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Shutdown)
	return c
}

func TestHandleTestSoulseekConnection_NoCore(t *testing.T) {
	s := NewServer(ServerConfig{Config: &core.Config{}})

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/sources/soulseek/test", map[string]interface{}{
		"username": "user",
		"password": "pass",
	}, &body)

	if resp.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusServiceUnavailable)
	}
	if body["error"] != "core not initialised" {
		t.Errorf("error = %v", body["error"])
	}
}

func TestHandleTestSoulseekConnection_BadBody(t *testing.T) {
	c := soulseekCore(t, false)
	s := NewServer(ServerConfig{Config: &core.Config{}, Core: c})

	resp := doRequest(t, s, "POST", "/api/sources/soulseek/test", "not an object", nil)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
}

func TestSoulseekStatusRouteRemoved(t *testing.T) {
	s := NewServer(ServerConfig{Config: &core.Config{}})
	resp := doRequest(t, s, "GET", "/api/sources/soulseek/status", nil, nil)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusNotFound)
	}
}

func TestHealthRoute_SoulseekRow(t *testing.T) {
	cases := []struct {
		name    string
		enabled bool
		status  string
	}{
		{"disabled", false, "dead"},
		{"enabled", true, "untested"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := soulseekCore(t, tc.enabled)
			s := NewServer(ServerConfig{Config: &core.Config{SoulseekEnabled: tc.enabled}, Core: c})

			var found *core.SourceHealth
			for _, h := range decodeBody[[]core.SourceHealth](t, send(t, s, "GET", "/api/sources/health", nil, "")) {
				if h.Name == "soulseek" {
					h := h
					found = &h
				}
			}
			if found == nil || found.Status != tc.status {
				t.Fatalf("soulseek row = %+v, want status %s", found, tc.status)
			}
		})
	}
}
