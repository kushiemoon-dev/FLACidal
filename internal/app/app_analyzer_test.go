package app

import (
	"os"
	"path/filepath"
	"testing"
)

// Characterization tests for the "Analyzer Methods" section of app.go.

func TestAnalyzeFile_InvalidFile(t *testing.T) {
	a := &App{} // nil logBuffer — the Info() call only runs on success
	path := filepath.Join(t.TempDir(), "not-a-real-flac.flac")
	if err := os.WriteFile(path, []byte("nope"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := a.AnalyzeFile(path); err == nil {
		t.Error("AnalyzeFile() on invalid FLAC content: want error, got nil")
	}
}

func TestAnalyzeMultiple_Empty(t *testing.T) {
	a := &App{}
	got := a.AnalyzeMultiple(nil)
	if len(got) != 0 {
		t.Errorf("AnalyzeMultiple(nil) = %v, want empty", got)
	}
}

func TestQuickAnalyze_InvalidFile(t *testing.T) {
	a := &App{}
	if _, err := a.QuickAnalyze(filepath.Join(t.TempDir(), "does-not-exist.flac")); err == nil {
		t.Error("QuickAnalyze() on a missing file: want error, got nil")
	}
}
