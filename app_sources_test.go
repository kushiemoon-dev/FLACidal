package main

import (
	"os"
	"path/filepath"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Characterization tests for the "Source Manager Methods" section of app.go,
// plus GetSourceHealth/poolSnapshotStatus/InstallSldl/GetSldlStatus/
// TestSoulseekConnection, which share app.go's "Converter Methods" comment
// header in the original file despite being about source health/Soulseek
// (see app_converter_test.go for the actual FFmpeg/conversion methods).
//
// NOT tested here (documented, not fixed):
//   - InstallSldl: downloads a real sldl binary over the network and emits
//     progress via runtime.EventsEmit, requiring a real Wails runtime context.
//   - TestSoulseekConnection's "binary present" branches: spawn a real sldl
//     process with a 30s timeout. Only the "binary not found" early return
//     (before any process is spawned) is exercised.
//   - FetchContentFromURL / GetSourceTrack / GetSourceAlbum / GetSourcePlaylist
//     success paths, ExpandDiscographyURL / QueueDiscographyAlbums success
//     paths: all require live network access to a registered source. Only
//     nil-guard and "not found"/"not detected" branches are exercised.

func TestGetSourceHealth(t *testing.T) {
	t.Run("all nil", func(t *testing.T) {
		a := &App{}
		if got := a.GetSourceHealth(); len(got) != 0 {
			t.Errorf("GetSourceHealth() with no sources = %v, want empty", got)
		}
	})

	t.Run("soulseek: no binary, no credentials -> reason reflects current priority", func(t *testing.T) {
		a := &App{
			soulseekSource: core.NewSoulseekSource(filepath.Join(t.TempDir(), "does-not-exist"), "", ""),
			config:         &core.Config{},
		}
		got := a.GetSourceHealth()
		if len(got) != 1 {
			t.Fatalf("GetSourceHealth() = %v, want exactly the soulseek entry", got)
		}
		if got[0].Name != "soulseek" || got[0].Status != "dead" {
			t.Errorf("GetSourceHealth() soulseek entry = %+v, want Status=dead", got[0])
		}
		// Current behavior: the credentials check runs before the binary-missing
		// check, so an empty username/password reports "credentials not
		// configured" even though the binary is also missing.
		if got[0].Reason != "credentials not configured" {
			t.Errorf("GetSourceHealth() soulseek reason = %q, want %q (current priority order)", got[0].Reason, "credentials not configured")
		}
	})
}

func TestPoolSnapshotStatus(t *testing.T) {
	tests := []struct {
		name string
		in   []core.EndpointStat
		want string
	}{
		{"empty", nil, "untested"},
		{"all live", []core.EndpointStat{{State: "live"}, {State: "probation"}}, "online"},
		{"some dead", []core.EndpointStat{{State: "live"}, {State: "dead"}}, "degraded"},
		{"all dead", []core.EndpointStat{{State: "dead"}, {State: "dead"}}, "dead"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := poolSnapshotStatus(tt.in); got != tt.want {
				t.Errorf("poolSnapshotStatus(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestGetSldlStatus(t *testing.T) {
	t.Run("binary missing", func(t *testing.T) {
		a := &App{config: &core.Config{SoulseekBinaryPath: filepath.Join(t.TempDir(), "does-not-exist")}}
		got := a.GetSldlStatus()
		if got["installed"] != false {
			t.Errorf("GetSldlStatus() with missing binary = %v, want installed=false", got)
		}
	})
	t.Run("binary present but not a real sldl build", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "sldl")
		if err := os.WriteFile(path, []byte("not an executable"), 0755); err != nil {
			t.Fatalf("setup: %v", err)
		}
		a := &App{config: &core.Config{SoulseekBinaryPath: path}}
		got := a.GetSldlStatus()
		if got["installed"] != true {
			t.Errorf("GetSldlStatus() with a present file = %v, want installed=true", got)
		}
		if got["version"] != "" {
			t.Errorf("GetSldlStatus() version = %q, want empty (garbage binary can't run --version)", got["version"])
		}
	})
}

func TestTestSoulseekConnection_BinaryNotFound(t *testing.T) {
	a := &App{config: &core.Config{SoulseekBinaryPath: filepath.Join(t.TempDir(), "does-not-exist")}}
	got := a.TestSoulseekConnection("user", "pass")
	if got["success"] != false || got["message"] != "sldl not found" {
		t.Errorf("TestSoulseekConnection() with missing binary = %v", got)
	}
}

func TestGetAvailableSources(t *testing.T) {
	a := &App{sourceManager: core.NewSourceManager()}
	if got := a.GetAvailableSources(); len(got) != 0 {
		t.Errorf("GetAvailableSources() with no registered sources = %v, want empty", got)
	}
}

func TestGetPreferredSource_DefaultsToTidal(t *testing.T) {
	a := &App{sourceManager: core.NewSourceManager()}
	if got := a.GetPreferredSource(); got != "tidal" {
		t.Errorf("GetPreferredSource() with no registered sources = %q, want %q (fallback)", got, "tidal")
	}
}

func TestSetPreferredSource(t *testing.T) {
	a := &App{sourceManager: core.NewSourceManager()} // nil logBuffer is nil-guarded
	a.SetPreferredSource("qobuz")
	got, ok := a.sourceManager.GetPreferredSource()
	if ok {
		t.Errorf("SetPreferredSource() then GetSource: got a source %v, want none registered", got)
	}
}

func TestDetectSourceFromURL_NoSourcesRegistered(t *testing.T) {
	a := &App{sourceManager: core.NewSourceManager()}
	got := a.DetectSourceFromURL("https://tidal.com/browse/track/12345")
	if got["detected"] != false {
		t.Errorf("DetectSourceFromURL() with no registered sources = %v, want detected=false", got)
	}
}

func TestFetchContentFromURL_NoSourcesRegistered(t *testing.T) {
	a := &App{sourceManager: core.NewSourceManager()}
	if _, err := a.FetchContentFromURL("https://tidal.com/browse/track/12345"); err == nil {
		t.Error("FetchContentFromURL() with no registered sources: want error, got nil")
	}
}

func TestExpandDiscographyURL(t *testing.T) {
	t.Run("not a discography URL", func(t *testing.T) {
		a := &App{}
		if _, err := a.ExpandDiscographyURL("https://example.com/not-spotify"); err == nil {
			t.Error("ExpandDiscographyURL() with a non-Spotify URL: want error, got nil")
		}
	})
	t.Run("valid discography URL but nil spotifySearch", func(t *testing.T) {
		a := &App{}
		_, err := a.ExpandDiscographyURL("https://open.spotify.com/artist/6l3HvQ5sa6mXTsMTB19rO5/discography/album")
		if err == nil {
			t.Error("ExpandDiscographyURL() with nil spotifySearch: want error, got nil")
		}
	})
}

func TestQueueDiscographyAlbums_Guards(t *testing.T) {
	t.Run("nil downloadManager", func(t *testing.T) {
		a := &App{}
		if _, err := a.QueueDiscographyAlbums(nil, "/tmp/out"); err == nil {
			t.Error("QueueDiscographyAlbums() with nil downloadManager: want error, got nil")
		}
	})
	t.Run("empty outputDir", func(t *testing.T) {
		a := &App{downloadManager: core.NewDownloadManager(core.NewTidalHifiService(), 1)}
		if _, err := a.QueueDiscographyAlbums(nil, ""); err == nil {
			t.Error("QueueDiscographyAlbums() with empty outputDir: want error, got nil")
		}
	})
	t.Run("nil spotifySearch", func(t *testing.T) {
		a := &App{downloadManager: core.NewDownloadManager(core.NewTidalHifiService(), 1)}
		if _, err := a.QueueDiscographyAlbums(nil, "/tmp/out"); err == nil {
			t.Error("QueueDiscographyAlbums() with nil spotifySearch: want error, got nil")
		}
	})
}

func TestGetSourceTrack_NotFound(t *testing.T) {
	a := &App{sourceManager: core.NewSourceManager()}
	if _, err := a.GetSourceTrack("unknown", "1"); err == nil {
		t.Error("GetSourceTrack() for an unregistered source: want error, got nil")
	}
}

func TestGetSourceAlbum_NotFound(t *testing.T) {
	a := &App{sourceManager: core.NewSourceManager()}
	if _, err := a.GetSourceAlbum("unknown", "1"); err == nil {
		t.Error("GetSourceAlbum() for an unregistered source: want error, got nil")
	}
}

func TestGetSourcePlaylist_NotFound(t *testing.T) {
	a := &App{sourceManager: core.NewSourceManager()}
	if _, err := a.GetSourcePlaylist("unknown", "1"); err == nil {
		t.Error("GetSourcePlaylist() for an unregistered source: want error, got nil")
	}
}

func TestUpdateQobuzCredentials(t *testing.T) {
	core.SetDataDir(t.TempDir())
	a := &App{qobuzSource: core.NewQobuzSource("", ""), config: &core.Config{}}

	if err := a.UpdateQobuzCredentials("app-id", "app-secret", "auth-token"); err != nil {
		t.Fatalf("UpdateQobuzCredentials() error = %v", err)
	}
	if a.config.QobuzAppID != "app-id" || !a.config.QobuzEnabled {
		t.Errorf("UpdateQobuzCredentials() did not persist to config: %+v", a.config)
	}
}

func TestUpdateQobuzCredentials_NilQobuzSourceSelfInitializes(t *testing.T) {
	core.SetDataDir(t.TempDir())
	a := &App{}
	if err := a.UpdateQobuzCredentials("id", "secret", "token"); err != nil {
		t.Fatalf("UpdateQobuzCredentials() with nil a.qobuzSource: unexpected error %v", err)
	}
	if a.qobuzSource == nil {
		t.Error("UpdateQobuzCredentials() did not self-initialize a.qobuzSource")
	}
	if a.config == nil || a.config.QobuzAppID != "id" {
		t.Error("UpdateQobuzCredentials() did not self-initialize a.config")
	}
}

func TestIsQobuzConfigured(t *testing.T) {
	// Note: QobuzSource.IsAvailable() short-circuits true when appID+appSecret+
	// userAuthToken are all set; otherwise it falls back to endpoint-pool
	// liveness (true by default on a freshly constructed pool, regardless of
	// credentials). So only the fully-configured case is deterministic here —
	// asserting "no credentials -> false" would depend on that pool's default
	// liveness state, not on credentials, and would be a flaky characterization.
	t.Run("fully configured (appID+secret+authToken)", func(t *testing.T) {
		qs := core.NewQobuzSource("app-id", "app-secret")
		qs.SetCredentials("app-id", "app-secret", "auth-token")
		a := &App{qobuzSource: qs}
		if !a.IsQobuzConfigured() {
			t.Error("IsQobuzConfigured() with appID+secret+authToken set = false, want true")
		}
	})
	t.Run("nil qobuzSource returns false", func(t *testing.T) {
		a := &App{}
		if a.IsQobuzConfigured() {
			t.Error("IsQobuzConfigured() with nil a.qobuzSource = true, want false")
		}
	})
}
