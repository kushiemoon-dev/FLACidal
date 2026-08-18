package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeFile_InvalidFile(t *testing.T) {
	a := &App{} // logBuffer left nil — Info() is only reached on the success path
	path := filepath.Join(t.TempDir(), "not-a-real-flac.flac")
	if err := os.WriteFile(path, []byte("nope"), 0644); err != nil {
		t.Fatalf("test setup failed: %v", err)
	}
	if _, err := a.AnalyzeFile(path); err == nil {
		t.Error("AnalyzeFile() with bogus FLAC content: expected an error, got nil")
	}
}

func TestAnalyzeMultiple_Empty(t *testing.T) {
	a := &App{}
	got := a.AnalyzeMultiple(nil)
	if len(got) != 0 {
		t.Errorf("AnalyzeMultiple(nil) = %v, expected empty result", got)
	}
}

func TestQuickAnalyze_InvalidFile(t *testing.T) {
	a := &App{}
	if _, err := a.QuickAnalyze(filepath.Join(t.TempDir(), "does-not-exist.flac")); err == nil {
		t.Error("QuickAnalyze() on a nonexistent file: expected an error, got nil")
	}
}
