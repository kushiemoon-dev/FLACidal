package api

import (
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

type trackHistoryBody struct {
	Entries []core.HistoryEntry `json:"entries"`
	Total   int                 `json:"total"`
}

func TestHandleGetTrackHistory_EmptyIsArrayNotNull(t *testing.T) {
	s := newTestServerWithDB(t)

	var raw map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/track-history", nil, &raw)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	entries, ok := raw["entries"].([]interface{})
	if !ok || len(entries) != 0 {
		t.Errorf("entries = %#v, want an empty array", raw["entries"])
	}
}

func TestHandleGetTrackHistory_FiltersByStatus(t *testing.T) {
	s := newTestServerWithDB(t)
	for _, e := range []core.HistoryEntry{
		{Title: "bad", Status: "failed", ErrorMessage: "boom"},
		{Title: "good", Status: "completed"},
	} {
		if err := s.db.InsertHistoryEntry(e); err != nil {
			t.Fatal(err)
		}
	}

	var body trackHistoryBody
	resp := doRequest(t, s, "GET", "/api/track-history?status=failed", nil, &body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if body.Total != 1 || len(body.Entries) != 1 || body.Entries[0].Title != "bad" {
		t.Errorf("body = %+v, want only the failed entry", body)
	}
}

func TestHandleGetTrackHistory_InvalidStatusIs400(t *testing.T) {
	s := newTestServerWithDB(t)

	resp := doRequest(t, s, "GET", "/api/track-history?status=bogus", nil, nil)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestHandleGetTrackHistory_LimitIsCapped(t *testing.T) {
	s := newTestServerWithDB(t)
	for i := 0; i < 205; i++ {
		if err := s.db.InsertHistoryEntry(core.HistoryEntry{Title: "t", Status: "failed"}); err != nil {
			t.Fatal(err)
		}
	}

	var body trackHistoryBody
	doRequest(t, s, "GET", "/api/track-history?limit=999", nil, &body)

	if len(body.Entries) > 200 {
		t.Errorf("got %d entries, limit should cap at 200", len(body.Entries))
	}
	if body.Total != 205 {
		t.Errorf("total = %d, want 205", body.Total)
	}
}

func TestHandleGetHistory_EmptyIsArray(t *testing.T) {
	s := newTestServerWithDB(t)

	var raw []interface{}
	resp := doRequest(t, s, "GET", "/api/history", nil, &raw)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if raw == nil || len(raw) != 0 {
		t.Errorf("body = %#v, want []", raw)
	}
}
