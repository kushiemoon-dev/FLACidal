package api

import (
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Tests for GET /api/history/recent.

func TestHandleGetRecentAlbums_NoDB(t *testing.T) {
	s := newTestServer(t)

	var albums []map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/history/recent", nil, &albums)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if len(albums) != 0 {
		t.Errorf("albums = %v, want empty (no DB configured)", albums)
	}
}

func TestHandleGetRecentAlbums_ReshapesToSnakeCase(t *testing.T) {
	s := newTestServerWithDB(t)

	if err := s.db.SaveDownloadRecord(&core.DownloadRecord{TidalContentID: "1", TidalContentName: "Daft Punk — Discovery", ContentType: "album", TracksTotal: 14}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	var albums []map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/history/recent?limit=24", nil, &albums)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if len(albums) != 1 {
		t.Fatalf("albums = %v, want 1 entry", albums)
	}
	if albums[0]["title"] != "Discovery" || albums[0]["artist"] != "Daft Punk" || albums[0]["content_id"] != "1" {
		t.Errorf("albums[0] = %v, want title=Discovery artist=Daft Punk content_id=1", albums[0])
	}
}

func TestHandleGetRecentAlbums_RespectsLimit(t *testing.T) {
	s := newTestServerWithDB(t)

	for _, id := range []string{"1", "2", "3"} {
		if err := s.db.SaveDownloadRecord(&core.DownloadRecord{TidalContentID: id, TidalContentName: "Artist — Album " + id, ContentType: "album"}); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	var albums []map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/history/recent?limit=2", nil, &albums)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if len(albums) != 2 {
		t.Errorf("albums = %d entries, want 2 (limit)", len(albums))
	}
}
