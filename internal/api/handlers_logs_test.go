package api

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	core "github.com/kushiemoon-dev/flacidal-core"
)

func TestHandleGetLogPath(t *testing.T) {
	s := newTestServer(t)

	var body map[string]string
	resp := doRequest(t, s, "GET", "/api/logs/path", nil, &body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if body["path"] != core.GetLogFilePath() {
		t.Errorf("path = %q, want %q", body["path"], core.GetLogFilePath())
	}
}
