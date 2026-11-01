package api

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func extensionServer(t *testing.T, token string) *Server {
	t.Helper()
	c := soulseekCore(t, false)
	return newServerWith(t, func(cfg *ServerConfig) {
		cfg.Core = c
		cfg.AuthToken = token
	})
}

func TestExtensionRoutes_RequireToken(t *testing.T) {
	s := extensionServer(t, "s3cret")
	routes := []struct{ method, path string }{
		{"GET", "/api/extensions"},
		{"POST", "/api/extensions/install"},
		{"POST", "/api/extensions/install/registry"},
		{"POST", "/api/extensions/demo/uninstall"},
		{"POST", "/api/extensions/demo/enabled"},
		{"POST", "/api/extensions/demo/auth"},
		{"GET", "/api/extensions/registries"},
		{"POST", "/api/extensions/registries"},
		{"POST", "/api/extensions/registries/remove"},
		{"POST", "/api/extensions/registry"},
		{"GET", "/api/sources/order"},
	}
	for _, r := range routes {
		if got := send(t, s, r.method, r.path, nil, "{}").StatusCode; got != fiber.StatusUnauthorized {
			t.Errorf("%s %s without token = %d, want 401", r.method, r.path, got)
		}
	}
}

func TestExtensionRoutes_ListAndCodedErrors(t *testing.T) {
	s := extensionServer(t, "")

	var list []core.ExtensionInfo
	if resp := doRequest(t, s, "GET", "/api/extensions", nil, &list); resp.StatusCode != fiber.StatusOK || len(list) != 0 {
		t.Fatalf("list: status=%d list=%v", resp.StatusCode, list)
	}

	var body map[string]string
	resp := doRequest(t, s, "POST", "/api/extensions/install", map[string]string{"url": "http://example.com/e.zip"}, &body)
	if resp.StatusCode != fiber.StatusBadRequest || !strings.HasPrefix(body["error"], "[extension.insecure_url]") {
		t.Fatalf("install http: status=%d body=%v", resp.StatusCode, body)
	}

	resp = doRequest(t, s, "POST", "/api/extensions/missing/enabled", map[string]bool{"enabled": true}, &body)
	if resp.StatusCode != fiber.StatusBadRequest || !strings.HasPrefix(body["error"], "[extension.not_found]") {
		t.Fatalf("enable missing: status=%d body=%v", resp.StatusCode, body)
	}

	var regs []string
	doRequest(t, s, "GET", "/api/extensions/registries", nil, &regs)
	if len(regs) != 1 || regs[0] != core.DefaultRegistrySource {
		t.Fatalf("registries = %v", regs)
	}
	resp = doRequest(t, s, "POST", "/api/extensions/registries", map[string]string{"url": "https://evil.example/a/b"}, &body)
	if resp.StatusCode != fiber.StatusBadRequest || !strings.HasPrefix(body["error"], "[extension.registry]") {
		t.Fatalf("bad registry: status=%d body=%v", resp.StatusCode, body)
	}

	var order []string
	doRequest(t, s, "GET", "/api/sources/order", nil, &order)
	if len(order) == 0 || order[len(order)-1] != "soulseek" {
		t.Fatalf("effective order = %v", order)
	}
}

func TestExtensionRoutes_NoCore(t *testing.T) {
	s := NewServer(ServerConfig{Config: &core.Config{}})
	resp := send(t, s, "GET", "/api/extensions", nil, "")
	b, _ := io.ReadAll(resp.Body)
	var body map[string]string
	_ = json.Unmarshal(b, &body)
	if resp.StatusCode != fiber.StatusServiceUnavailable || body["error"] != "core unavailable" {
		t.Fatalf("status=%d body=%s", resp.StatusCode, b)
	}
}

func TestExtensionRoutes_UnknownRoute(t *testing.T) {
	s := extensionServer(t, "")
	if got := send(t, s, "GET", "/api/extensions/nope/whatever", nil, "").StatusCode; got != fiber.StatusNotFound {
		t.Fatalf("unknown route = %d, want 404", got)
	}
}
