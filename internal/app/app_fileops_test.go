package app

import (
	"os"
	"path/filepath"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Characterization tests for the "File Browser Methods" section of app.go.

func TestListDownloadedFiles(t *testing.T) {
	t.Run("no download folder configured", func(t *testing.T) {
		a := &App{}
		got, err := a.ListDownloadedFiles()
		if err != nil {
			t.Fatalf("ListDownloadedFiles() error = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("ListDownloadedFiles() with no folder = %v, want empty", got)
		}
	})
	t.Run("empty download folder", func(t *testing.T) {
		core.SetDataDir(t.TempDir())
		a := &App{config: &core.Config{DownloadFolder: t.TempDir()}}
		got, err := a.ListDownloadedFiles()
		if err != nil {
			t.Fatalf("ListDownloadedFiles() error = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("ListDownloadedFiles() on an empty folder = %v, want empty", got)
		}
	})
}

func TestDeleteFile(t *testing.T) {
	a := &App{}
	t.Run("rejects non-flac files", func(t *testing.T) {
		if err := a.DeleteFile("/tmp/not-a-flac.mp3"); err == nil {
			t.Error("DeleteFile() on a non-.flac path: want error, got nil")
		}
	})
	t.Run("deletes a real .flac file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "song.flac")
		if err := os.WriteFile(path, []byte("fake"), 0644); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if err := a.DeleteFile(path); err != nil {
			t.Fatalf("DeleteFile() error = %v", err)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Error("DeleteFile() did not remove the file")
		}
	})
}

func TestGetFileMetadata_InvalidFile(t *testing.T) {
	a := &App{}
	path := filepath.Join(t.TempDir(), "not-a-real-flac.flac")
	if err := os.WriteFile(path, []byte("not actually flac data"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := a.GetFileMetadata(path); err == nil {
		t.Error("GetFileMetadata() on invalid FLAC content: want error, got nil")
	}
}

func TestGetFileCoverArt_InvalidFile(t *testing.T) {
	a := &App{}
	path := filepath.Join(t.TempDir(), "not-a-real-flac.flac")
	if err := os.WriteFile(path, []byte("not actually flac data"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := a.GetFileCoverArt(path); err == nil {
		t.Error("GetFileCoverArt() on invalid FLAC content: want error, got nil")
	}
}

func TestGetRenameTemplates(t *testing.T) {
	a := &App{}
	got := a.GetRenameTemplates()
	if len(got) == 0 {
		t.Error("GetRenameTemplates() = empty, want at least one built-in template")
	}
}

func TestPreviewRename(t *testing.T) {
	a := &App{}
	got := a.PreviewRename([]string{"/music/old.flac"}, "{title}")
	if len(got) != 1 {
		t.Fatalf("PreviewRename() returned %d entries, want 1", len(got))
	}
}

func TestRenameFiles(t *testing.T) {
	a := &App{} // nil logBuffer — the Info() call is nil-guarded
	dir := t.TempDir()
	path := filepath.Join(dir, "old.flac")
	if err := os.WriteFile(path, []byte("fake"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	got := a.RenameFiles([]string{path}, "{title}")
	if len(got) != 1 {
		t.Fatalf("RenameFiles() returned %d results, want 1", len(got))
	}
}
