package app

import (
	"os"
	"path/filepath"
	"testing"
)

// Characterization tests for the "Lyrics Methods" section of app.go.
//
// NOT tested here (documented, not fixed):
//   - FetchLyrics: makes a live network call to LRCLIB with no injectable
//     HTTP seam.
//   - FetchLyricsForFile / FetchAndEmbedLyrics / FetchAndEmbedLyricsMultiple
//     success paths: all reach the same LRCLIB network call once metadata
//     reads succeed. Only their fail-fast "invalid file" branches (which
//     return before any network call, since core.ReadFLACMetadata errors
//     first) are exercised.

func TestFetchLyricsForFile_InvalidFile(t *testing.T) {
	a := &App{}
	path := filepath.Join(t.TempDir(), "not-a-real-flac.flac")
	if err := os.WriteFile(path, []byte("nope"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := a.FetchLyricsForFile(path); err == nil {
		t.Error("FetchLyricsForFile() on invalid FLAC content: want error, got nil")
	}
}

func TestEmbedLyricsToFile_InvalidFile(t *testing.T) {
	a := &App{} // nil logBuffer — the Error() call on failure is nil-guarded
	path := filepath.Join(t.TempDir(), "not-a-real-flac.flac")
	if err := os.WriteFile(path, []byte("nope"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := a.EmbedLyricsToFile(path, "plain lyrics", ""); err == nil {
		t.Error("EmbedLyricsToFile() on invalid FLAC content: want error, got nil")
	}
}

func TestFetchAndEmbedLyrics_InvalidFile(t *testing.T) {
	a := &App{}
	path := filepath.Join(t.TempDir(), "not-a-real-flac.flac")
	if err := os.WriteFile(path, []byte("nope"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := a.FetchAndEmbedLyrics(path); err == nil {
		t.Error("FetchAndEmbedLyrics() on invalid FLAC content: want error, got nil")
	}
}

func TestFetchAndEmbedLyricsMultiple_InvalidFiles(t *testing.T) {
	a := &App{}
	dir := t.TempDir()
	path := filepath.Join(dir, "not-a-real-flac.flac")
	if err := os.WriteFile(path, []byte("nope"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	got := a.FetchAndEmbedLyricsMultiple([]string{path})
	if len(got) != 1 {
		t.Fatalf("FetchAndEmbedLyricsMultiple() returned %d results, want 1", len(got))
	}
	if got[0]["success"] != false {
		t.Errorf("FetchAndEmbedLyricsMultiple() on an invalid file = %v, want success=false", got[0])
	}
	if _, ok := got[0]["error"]; !ok {
		t.Errorf("FetchAndEmbedLyricsMultiple() = %v, want an 'error' key", got[0])
	}
}
