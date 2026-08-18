package api

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func TestServer_ServesFrontend_WhenDistBuilt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>flacidal</html>"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	s := NewServer(ServerConfig{
		Config:      &core.Config{},
		FrontendDir: dir,
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := s.app.Test(req, -1)
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
}

func TestServer_HelpfulError_WhenDistMissing(t *testing.T) {
	dir := t.TempDir()

	s := NewServer(ServerConfig{
		Config:      &core.Config{},
		FrontendDir: dir,
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := s.app.Test(req, -1)
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	if resp.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusServiceUnavailable)
	}

	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])
	if !strings.Contains(body, "npm run build") {
		t.Errorf("body = %q, want a hint to run 'npm run build'", body)
	}
}

func TestServer_DefaultFrontendDir(t *testing.T) {
	s := NewServer(ServerConfig{Config: &core.Config{}})
	if s.frontendDir != "frontend/dist" {
		t.Errorf("frontendDir = %q, want default %q", s.frontendDir, "frontend/dist")
	}
}
