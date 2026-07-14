package api

import (
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Tests for POST /api/history/refetch/:id.
//
// Mirrors internal/app's TestRefetchFromHistory. Only the "no db", "not
// found" and "unknown content type" branches are exercised — the "known
// content type" success path calls fetchContentByURL, which makes a live
// network call with no injectable HTTP seam (same documented limitation as
// handlers_search_test.go).

func newTestServerWithDB(t *testing.T) *Server {
	t.Helper()
	core.SetDataDir(t.TempDir())
	db, err := core.NewDatabase()
	if err != nil {
		t.Fatalf("core.NewDatabase: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	return NewServer(ServerConfig{
		Config: &core.Config{},
		DB:     db,
	})
}

func TestHandleRefetchFromHistory_NoDB(t *testing.T) {
	s := newTestServer(t)

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/history/refetch/x", nil, &body)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusInternalServerError)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleRefetchFromHistory_NotFound(t *testing.T) {
	s := newTestServerWithDB(t)

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/history/refetch/does-not-exist", nil, &body)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusNotFound)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}

func TestHandleRefetchFromHistory_UnknownContentType(t *testing.T) {
	s := newTestServerWithDB(t)

	if err := s.db.SaveDownloadRecord(&core.DownloadRecord{TidalContentID: "1", TidalContentName: "X", ContentType: "mix"}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/history/refetch/1", nil, &body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("body = %v, want an 'error' key", body)
	}
}
