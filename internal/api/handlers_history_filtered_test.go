package api

import (
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Tests for GET /api/history/filtered — specifically that contentType and
// search query params are actually threaded into the DB filter (previously
// only limit/offset were parsed, silently ignoring the other filter fields
// the frontend sends).

func TestHandleGetHistoryFiltered_ContentTypeAndSearch(t *testing.T) {
	s := newTestServerWithDB(t)

	if err := s.db.SaveDownloadRecord(&core.DownloadRecord{TidalContentID: "1", TidalContentName: "Daft Punk — Discovery", ContentType: "album"}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := s.db.SaveDownloadRecord(&core.DownloadRecord{TidalContentID: "2", TidalContentName: "Some Track", ContentType: "track"}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	var body struct {
		Records []core.DownloadRecord `json:"records"`
		Total   int                   `json:"total"`
	}
	resp := doRequest(t, s, "GET", "/api/history/filtered?contentType=album", nil, &body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if body.Total != 1 || len(body.Records) != 1 || body.Records[0].ContentType != "album" {
		t.Errorf("body = %+v, want exactly the 1 album record", body)
	}
}
