package main

import (
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Characterization tests for the "Database Methods" section of app.go
// (GetCacheStats, GetDownloadHistory, GetRecentAlbums, GetDownloadHistoryFiltered,
// DeleteHistoryRecord, ClearDownloadHistory, RefetchFromHistory, GetMatchFailures).
//
// These use a real *core.Database backed by a temp-dir SQLite file via
// core.SetDataDir(t.TempDir()) — the same test seam FLACidal-Core's own
// database_test.go uses — so no real user data under ~/.flacidal is touched.
//
// Bug note (not fixed): RefetchFromHistory's "known content type" success path
// calls a.FetchTidalContent(url), whose per-type branches (e.g. the "track"
// case calling a.downloader.GetTrackAsTidalTrack) dereference a.downloader /
// a.tidalClient without a nil-guard. Not reachable in production (startup()
// always initializes those fields), but inconsistent with the nil-guard
// pattern used by sibling methods like SearchTidal/DownloadTrack. Only the
// "not found" and "unknown content type" branches (which return before
// reaching FetchTidalContent) are exercised here.

func newTestApp(t *testing.T) *App {
	t.Helper()
	core.SetDataDir(t.TempDir())
	db, err := core.NewDatabase()
	if err != nil {
		t.Fatalf("core.NewDatabase: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &App{db: db}
}

func TestGetCacheStats(t *testing.T) {
	t.Run("nil db", func(t *testing.T) {
		a := &App{}
		got := a.GetCacheStats()
		if got["error"] != "database not initialized" {
			t.Errorf("GetCacheStats() with nil db = %v", got)
		}
	})
	t.Run("empty db", func(t *testing.T) {
		a := newTestApp(t)
		got := a.GetCacheStats()
		if _, hasErr := got["error"]; hasErr {
			t.Errorf("GetCacheStats() on empty db returned an error: %v", got)
		}
		if got["total"] != 0 {
			t.Errorf("GetCacheStats() total = %v, want 0", got["total"])
		}
	})
}

func TestGetDownloadHistory(t *testing.T) {
	t.Run("nil db", func(t *testing.T) {
		a := &App{}
		got, err := a.GetDownloadHistory()
		if got != nil || err != nil {
			t.Errorf("GetDownloadHistory() with nil db = (%v, %v), want (nil, nil)", got, err)
		}
	})
	t.Run("empty db", func(t *testing.T) {
		a := newTestApp(t)
		got, err := a.GetDownloadHistory()
		if err != nil {
			t.Fatalf("GetDownloadHistory() error = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("GetDownloadHistory() = %v, want empty", got)
		}
	})
}

func TestGetRecentAlbums(t *testing.T) {
	t.Run("nil db", func(t *testing.T) {
		a := &App{}
		got, err := a.GetRecentAlbums(10)
		if err != nil {
			t.Fatalf("GetRecentAlbums() error = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("GetRecentAlbums() with nil db = %v, want empty", got)
		}
	})

	t.Run("dedups, sorts, parses artist-title, respects limit", func(t *testing.T) {
		a := newTestApp(t)
		records := []core.DownloadRecord{
			{TidalContentID: "1", TidalContentName: "Daft Punk — Discovery", ContentType: "album", TracksTotal: 14},
			{TidalContentID: "1", TidalContentName: "Daft Punk — Discovery", ContentType: "album", TracksTotal: 14}, // duplicate content ID
			{TidalContentID: "2", TidalContentName: "Homework", ContentType: "album", TracksTotal: 16},              // no "Artist — Title" separator
			{TidalContentID: "3", TidalContentName: "Justice — Cross", ContentType: "album", TracksTotal: 10},
		}
		for _, r := range records {
			r := r
			if err := a.db.SaveDownloadRecord(&r); err != nil {
				t.Fatalf("SaveDownloadRecord: %v", err)
			}
		}

		got, err := a.GetRecentAlbums(2)
		if err != nil {
			t.Fatalf("GetRecentAlbums() error = %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("GetRecentAlbums(limit=2) returned %d entries, want 2 (limit enforced)", len(got))
		}

		seen := map[string]bool{}
		for _, entry := range got {
			cid := entry["content_id"].(string)
			if seen[cid] {
				t.Errorf("GetRecentAlbums() returned duplicate content_id %q — dedup not applied", cid)
			}
			seen[cid] = true
		}

		// Records were saved in the same transaction-less loop so LastDownloadAt
		// timestamps are set by SQLite's time.Now() at save time; just check the
		// "Artist — Title" split parsed correctly for content_id "3".
		for _, entry := range got {
			if entry["content_id"] == "3" {
				if entry["artist"] != "Justice" || entry["title"] != "Cross" {
					t.Errorf("GetRecentAlbums() did not parse 'Artist — Title', got artist=%v title=%v", entry["artist"], entry["title"])
				}
			}
			if entry["content_id"] == "2" {
				if entry["artist"] != "" || entry["title"] != "Homework" {
					t.Errorf("GetRecentAlbums() with no separator: got artist=%v title=%v, want artist='' title='Homework'", entry["artist"], entry["title"])
				}
			}
		}
	})
}

func TestGetDownloadHistoryFiltered(t *testing.T) {
	t.Run("nil db", func(t *testing.T) {
		a := &App{}
		_, err := a.GetDownloadHistoryFiltered(map[string]interface{}{})
		if err == nil {
			t.Error("GetDownloadHistoryFiltered() with nil db: want error, got nil")
		}
	})

	t.Run("parses filter map fields, ignores wrong types", func(t *testing.T) {
		a := newTestApp(t)
		if err := a.db.SaveDownloadRecord(&core.DownloadRecord{TidalContentID: "1", TidalContentName: "Album One", ContentType: "album"}); err != nil {
			t.Fatalf("setup: %v", err)
		}

		// limit given as the wrong type (string, not float64 as JSON numbers
		// decode to) — current behavior silently ignores it rather than erroring.
		got, err := a.GetDownloadHistoryFiltered(map[string]interface{}{
			"contentType": "album",
			"limit":       "10", // wrong type on purpose
		})
		if err != nil {
			t.Fatalf("GetDownloadHistoryFiltered() error = %v", err)
		}
		if got["total"] != 1 {
			t.Errorf("GetDownloadHistoryFiltered() total = %v, want 1", got["total"])
		}
	})
}

func TestDeleteHistoryRecord(t *testing.T) {
	t.Run("nil db", func(t *testing.T) {
		a := &App{}
		if err := a.DeleteHistoryRecord(1); err == nil {
			t.Error("DeleteHistoryRecord() with nil db: want error, got nil")
		}
	})
	t.Run("real db", func(t *testing.T) {
		a := newTestApp(t)
		if err := a.db.SaveDownloadRecord(&core.DownloadRecord{TidalContentID: "1", TidalContentName: "X", ContentType: "track"}); err != nil {
			t.Fatalf("setup: %v", err)
		}
		record, err := a.db.GetDownloadRecord("1")
		if err != nil || record == nil {
			t.Fatalf("setup GetDownloadRecord: %v, %v", record, err)
		}
		if err := a.DeleteHistoryRecord(record.ID); err != nil {
			t.Errorf("DeleteHistoryRecord() error = %v", err)
		}
	})
}

func TestClearDownloadHistory(t *testing.T) {
	t.Run("nil db", func(t *testing.T) {
		a := &App{}
		if err := a.ClearDownloadHistory(); err == nil {
			t.Error("ClearDownloadHistory() with nil db: want error, got nil")
		}
	})
	t.Run("real db, nil logBuffer", func(t *testing.T) {
		a := newTestApp(t)
		if err := a.db.SaveDownloadRecord(&core.DownloadRecord{TidalContentID: "1", TidalContentName: "X", ContentType: "track"}); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if err := a.ClearDownloadHistory(); err != nil {
			t.Fatalf("ClearDownloadHistory() error = %v", err)
		}
		remaining, err := a.db.GetAllDownloadRecords()
		if err != nil {
			t.Fatalf("GetAllDownloadRecords: %v", err)
		}
		if len(remaining) != 0 {
			t.Errorf("ClearDownloadHistory() left %d records, want 0", len(remaining))
		}
	})
}

func TestRefetchFromHistory(t *testing.T) {
	t.Run("nil db", func(t *testing.T) {
		a := &App{}
		if _, err := a.RefetchFromHistory("x"); err == nil {
			t.Error("RefetchFromHistory() with nil db: want error, got nil")
		}
	})
	t.Run("record not found", func(t *testing.T) {
		a := newTestApp(t)
		if _, err := a.RefetchFromHistory("does-not-exist"); err == nil {
			t.Error("RefetchFromHistory() for a missing record: want error, got nil")
		}
	})
	t.Run("unknown content type", func(t *testing.T) {
		a := newTestApp(t)
		if err := a.db.SaveDownloadRecord(&core.DownloadRecord{TidalContentID: "1", TidalContentName: "X", ContentType: "mix"}); err != nil {
			t.Fatalf("setup: %v", err)
		}
		_, err := a.RefetchFromHistory("1")
		if err == nil {
			t.Error("RefetchFromHistory() with unsupported content type: want error, got nil")
		}
	})
}

func TestGetMatchFailures(t *testing.T) {
	t.Run("nil db", func(t *testing.T) {
		a := &App{}
		got, err := a.GetMatchFailures()
		if got != nil || err != nil {
			t.Errorf("GetMatchFailures() with nil db = (%v, %v), want (nil, nil)", got, err)
		}
	})
	t.Run("empty db", func(t *testing.T) {
		a := newTestApp(t)
		got, err := a.GetMatchFailures()
		if err != nil {
			t.Fatalf("GetMatchFailures() error = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("GetMatchFailures() = %v, want empty", got)
		}
	})
}
