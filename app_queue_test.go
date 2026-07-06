package main

import (
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Characterization tests for the "Download Methods" section of app.go, plus
// the trailing queue-control methods (RetryDownload, RetryAllFailed,
// ExportFailedDownloads, CancelDownload, PauseDownloads, ResumeDownloads,
// IsQueuePaused) that live in the same file without their own header.
//
// NOT tested here (documented, not fixed):
//   - OpenFLACFilesDialog / SelectDownloadFolder: open real native file/folder
//     picker dialogs via the Wails runtime — requires a real runtime context.
//   - IsDownloaderAvailable's non-nil branch: core.TidalHifiService.IsAvailable()
//     makes a live network HEAD request with no injectable seam. Only the
//     nil-downloader branch is exercised.
//   - DownloadTrack / DownloadTrackFromTidal / QueueDownloads / QueueQobuzDownloads
//     / QueueArtistAlbum / DownloadArtistAssets / QueueSingleDownload success
//     paths: all require live network access (Tidal proxy / artist image URLs).
//     Only their nil-guard and empty-argument error branches are exercised.
//   - OpenDownloadFolder's success branch: calls runtime.BrowserOpenURL, which
//     requires a real Wails runtime context (see app_logging_test.go note on
//     runtime.EventsEmit — same log.Fatalf risk). Only the folder=="" error
//     branch is exercised.
//   - ExportFailedDownloads' non-empty-jobs branch: opens a real native save
//     dialog via runtime.SaveFileDialog. Only the "no failed jobs" early
//     return (which happens before the dialog) is exercised.

func TestGetDownloadFolder_SetDownloadFolder(t *testing.T) {
	core.SetDataDir(t.TempDir())

	a := &App{}
	if got := a.GetDownloadFolder(); got != "" {
		t.Errorf("GetDownloadFolder() with nil config = %q, want empty", got)
	}

	if err := a.SetDownloadFolder("/music/out"); err != nil {
		t.Fatalf("SetDownloadFolder() error = %v", err)
	}
	if a.config == nil {
		t.Fatal("SetDownloadFolder() did not self-initialize a.config")
	}
	if got := a.GetDownloadFolder(); got != "/music/out" {
		t.Errorf("GetDownloadFolder() = %q, want %q", got, "/music/out")
	}
}

func TestIsDownloaderAvailable_Nil(t *testing.T) {
	a := &App{}
	if a.IsDownloaderAvailable() {
		t.Error("IsDownloaderAvailable() with nil downloader = true, want false")
	}
}

func TestDownloadTrack_Guards(t *testing.T) {
	t.Run("nil downloader", func(t *testing.T) {
		a := &App{}
		if _, err := a.DownloadTrack(1, "/tmp/out"); err == nil {
			t.Error("DownloadTrack() with nil downloader: want error, got nil")
		}
	})
	t.Run("empty outputDir", func(t *testing.T) {
		a := &App{downloader: core.NewTidalHifiService()}
		if _, err := a.DownloadTrack(1, ""); err == nil {
			t.Error("DownloadTrack() with empty outputDir: want error, got nil")
		}
	})
}

func TestDownloadTrackFromTidal_Guards(t *testing.T) {
	t.Run("nil downloader", func(t *testing.T) {
		a := &App{}
		if _, err := a.DownloadTrackFromTidal(core.TidalTrack{ID: 1}, "/tmp/out"); err == nil {
			t.Error("DownloadTrackFromTidal() with nil downloader: want error, got nil")
		}
	})
	t.Run("empty outputDir", func(t *testing.T) {
		a := &App{downloader: core.NewTidalHifiService()}
		if _, err := a.DownloadTrackFromTidal(core.TidalTrack{ID: 1}, ""); err == nil {
			t.Error("DownloadTrackFromTidal() with empty outputDir: want error, got nil")
		}
	})
}

func TestQueueDownloads_Guards(t *testing.T) {
	t.Run("nil downloadManager", func(t *testing.T) {
		a := &App{}
		if _, err := a.QueueDownloads(nil, "/tmp/out", "name", "id", "album"); err == nil {
			t.Error("QueueDownloads() with nil downloadManager: want error, got nil")
		}
	})
	t.Run("empty outputDir", func(t *testing.T) {
		a := &App{downloadManager: core.NewDownloadManager(core.NewTidalHifiService(), 1)}
		if _, err := a.QueueDownloads(nil, "", "name", "id", "album"); err == nil {
			t.Error("QueueDownloads() with empty outputDir: want error, got nil")
		}
	})
}

func TestQueueQobuzDownloads_Guards(t *testing.T) {
	t.Run("nil downloadManager", func(t *testing.T) {
		a := &App{}
		if _, err := a.QueueQobuzDownloads(nil, "/tmp/out", "name"); err == nil {
			t.Error("QueueQobuzDownloads() with nil downloadManager: want error, got nil")
		}
	})
	t.Run("empty outputDir", func(t *testing.T) {
		a := &App{downloadManager: core.NewDownloadManager(core.NewTidalHifiService(), 1)}
		if _, err := a.QueueQobuzDownloads(nil, "", "name"); err == nil {
			t.Error("QueueQobuzDownloads() with empty outputDir: want error, got nil")
		}
	})
}

func TestQueueArtistAlbum_Guards(t *testing.T) {
	t.Run("nil downloadManager", func(t *testing.T) {
		a := &App{}
		if _, err := a.QueueArtistAlbum("album-id", "artist", "/tmp/out"); err == nil {
			t.Error("QueueArtistAlbum() with nil downloadManager: want error, got nil")
		}
	})
	t.Run("empty outputDir", func(t *testing.T) {
		a := &App{downloadManager: core.NewDownloadManager(core.NewTidalHifiService(), 1)}
		if _, err := a.QueueArtistAlbum("album-id", "artist", ""); err == nil {
			t.Error("QueueArtistAlbum() with empty outputDir: want error, got nil")
		}
	})
}

func TestDownloadArtistAssets_EmptyOutputDir(t *testing.T) {
	a := &App{}
	if _, err := a.DownloadArtistAssets("artist-id", "artist", ""); err == nil {
		t.Error("DownloadArtistAssets() with empty outputDir: want error, got nil")
	}
}

func TestQueueSingleDownload_Guards(t *testing.T) {
	t.Run("nil downloadManager", func(t *testing.T) {
		a := &App{}
		if err := a.QueueSingleDownload(1, "/tmp/out", "title", "artist"); err == nil {
			t.Error("QueueSingleDownload() with nil downloadManager: want error, got nil")
		}
	})
	t.Run("empty outputDir", func(t *testing.T) {
		a := &App{downloadManager: core.NewDownloadManager(core.NewTidalHifiService(), 1)}
		if err := a.QueueSingleDownload(1, "", "title", "artist"); err == nil {
			t.Error("QueueSingleDownload() with empty outputDir: want error, got nil")
		}
	})
}

func TestGetDownloadQueueStatus(t *testing.T) {
	t.Run("nil downloadManager", func(t *testing.T) {
		a := &App{}
		got := a.GetDownloadQueueStatus()
		if got["running"] != false {
			t.Errorf("GetDownloadQueueStatus() with nil manager = %v", got)
		}
	})
	t.Run("real, un-started downloadManager", func(t *testing.T) {
		a := &App{downloadManager: core.NewDownloadManager(core.NewTidalHifiService(), 1)}
		got := a.GetDownloadQueueStatus()
		if got["running"] != false || got["paused"] != false || got["activeCount"] != 0 || got["queueLength"] != 0 {
			t.Errorf("GetDownloadQueueStatus() on fresh manager = %v, want all-zero/false", got)
		}
	})
}

func TestGetDownloadOptions(t *testing.T) {
	t.Run("nil config returns hardcoded defaults", func(t *testing.T) {
		a := &App{}
		got := a.GetDownloadOptions()
		if got["quality"] != "LOSSLESS" || got["fileNameFormat"] != "{artist} - {title}" {
			t.Errorf("GetDownloadOptions() with nil config = %v", got)
		}
	})
	t.Run("config with blank quality/format falls back to defaults", func(t *testing.T) {
		a := &App{config: &core.Config{}}
		got := a.GetDownloadOptions()
		if got["quality"] != "LOSSLESS" || got["fileNameFormat"] != "{artist} - {title}" {
			t.Errorf("GetDownloadOptions() with blank config fields = %v, want fallback defaults", got)
		}
	})
	t.Run("config values pass through", func(t *testing.T) {
		a := &App{config: &core.Config{DownloadQuality: "HI_RES", FileNameFormat: "{title}"}}
		got := a.GetDownloadOptions()
		if got["quality"] != "HI_RES" || got["fileNameFormat"] != "{title}" {
			t.Errorf("GetDownloadOptions() = %v, want quality=HI_RES fileNameFormat={title}", got)
		}
	})
}

func TestSetDownloadOptions(t *testing.T) {
	core.SetDataDir(t.TempDir())
	a := &App{}
	err := a.SetDownloadOptions("HI_RES", "{title}", true, true, true, true)
	if err != nil {
		t.Fatalf("SetDownloadOptions() error = %v", err)
	}
	if a.config == nil {
		t.Fatal("SetDownloadOptions() did not self-initialize a.config")
	}
	if a.config.DownloadQuality != "HI_RES" || a.config.FileNameFormat != "{title}" {
		t.Errorf("SetDownloadOptions() did not persist to config: %+v", a.config)
	}
}

func TestOpenDownloadFolder_EmptyFolder(t *testing.T) {
	a := &App{}
	if err := a.OpenDownloadFolder(""); err == nil {
		t.Error("OpenDownloadFolder() with empty folder: want error, got nil")
	}
}

func TestRetryDownload_Guards(t *testing.T) {
	t.Run("nil downloadManager", func(t *testing.T) {
		a := &App{}
		if err := a.RetryDownload(1); err == nil {
			t.Error("RetryDownload() with nil downloadManager: want error, got nil")
		}
	})
	t.Run("no download folder configured", func(t *testing.T) {
		a := &App{downloadManager: core.NewDownloadManager(core.NewTidalHifiService(), 1)}
		if err := a.RetryDownload(1); err == nil {
			t.Error("RetryDownload() with no download folder: want error, got nil")
		}
	})
}

func TestRetryAllFailed(t *testing.T) {
	t.Run("nil downloadManager", func(t *testing.T) {
		a := &App{}
		if _, err := a.RetryAllFailed(); err == nil {
			t.Error("RetryAllFailed() with nil downloadManager: want error, got nil")
		}
	})
	t.Run("real manager, no failed jobs", func(t *testing.T) {
		a := &App{downloadManager: core.NewDownloadManager(core.NewTidalHifiService(), 1)}
		n, err := a.RetryAllFailed()
		if err != nil || n != 0 {
			t.Errorf("RetryAllFailed() = (%d, %v), want (0, nil)", n, err)
		}
	})
}

func TestExportFailedDownloads_NilManager(t *testing.T) {
	a := &App{}
	if _, err := a.ExportFailedDownloads("csv"); err == nil {
		t.Error("ExportFailedDownloads() with nil downloadManager: want error, got nil")
	}
}

func TestExportFailedDownloads_NoFailedJobs(t *testing.T) {
	a := &App{downloadManager: core.NewDownloadManager(core.NewTidalHifiService(), 1)}
	path, err := a.ExportFailedDownloads("csv")
	if err != nil || path != "" {
		t.Errorf("ExportFailedDownloads() with no failed jobs = (%q, %v), want (\"\", nil)", path, err)
	}
}

func TestCancelDownload_Guards(t *testing.T) {
	t.Run("nil downloadManager", func(t *testing.T) {
		a := &App{}
		if err := a.CancelDownload(1); err == nil {
			t.Error("CancelDownload() with nil downloadManager: want error, got nil")
		}
	})
	t.Run("track not active", func(t *testing.T) {
		a := &App{downloadManager: core.NewDownloadManager(core.NewTidalHifiService(), 1)}
		if err := a.CancelDownload(1); err == nil {
			t.Error("CancelDownload() for an inactive track: want error, got nil")
		}
	})
}

func TestPauseResumeQueue(t *testing.T) {
	t.Run("nil downloadManager", func(t *testing.T) {
		a := &App{}
		if a.PauseDownloads() {
			t.Error("PauseDownloads() with nil downloadManager = true, want false")
		}
		if a.ResumeDownloads() {
			t.Error("ResumeDownloads() with nil downloadManager = true, want false")
		}
		if a.IsQueuePaused() {
			t.Error("IsQueuePaused() with nil downloadManager = true, want false")
		}
	})

	t.Run("real manager, logBuffer nil to avoid the runtime.EventsEmit branch", func(t *testing.T) {
		a := &App{downloadManager: core.NewDownloadManager(core.NewTidalHifiService(), 1)}

		if !a.PauseDownloads() {
			t.Error("PauseDownloads() first call = false, want true")
		}
		if a.PauseDownloads() {
			t.Error("PauseDownloads() second call (already paused) = true, want false")
		}
		if !a.IsQueuePaused() {
			t.Error("IsQueuePaused() after pausing = false, want true")
		}
		if !a.ResumeDownloads() {
			t.Error("ResumeDownloads() first call = false, want true")
		}
		if a.ResumeDownloads() {
			t.Error("ResumeDownloads() second call (already running) = true, want false")
		}
		if a.IsQueuePaused() {
			t.Error("IsQueuePaused() after resuming = true, want false")
		}
	})
}
