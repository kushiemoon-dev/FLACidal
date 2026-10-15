package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func TestUnknownAPIRouteReturnsJSON404(t *testing.T) {
	s := newTestServer(t)

	var body map[string]string
	resp := doRequest(t, s, "GET", "/api/does-not-exist", nil, &body)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want JSON", ct)
	}
	if body["error"] != "not found" {
		t.Errorf("body = %v", body)
	}
}

func TestKnownAPIRouteIsNotShadowedByNotFound(t *testing.T) {
	s := newTestServer(t)

	resp := doRequest(t, s, "GET", "/api/version", nil, nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestSPAIsStillServedForNonAPIPaths(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>spa</html>"), 0644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(ServerConfig{Config: &core.Config{}, FrontendDir: dir})

	for _, path := range []string{"/", "/some/client/route"} {
		resp := doRequest(t, s, "GET", path, nil, nil)
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("GET %s status = %d, want 200", path, resp.StatusCode)
		}
	}
}
