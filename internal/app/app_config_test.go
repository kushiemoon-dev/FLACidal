package app

import (
	"os"
	"path/filepath"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Characterization tests for the "Config Methods" section of app.go
// (GetConfig, SaveConfig, SetSourceOrder, ResetToDefaults, GetConnectionStatus,
// CheckAPIStatus, endpointStatToStatus).
//
// NOT tested here (documented, not fixed):
//   - OpenConfigFolder / openFolder: spawns a real OS file-manager process
//     (xdg-open/open/explorer) as a side effect — unsafe to run in a test.

func TestGetConfig(t *testing.T) {
	cfg := &core.Config{Theme: "dark"}
	a := &App{config: cfg}
	if got := a.GetConfig(); got != cfg {
		t.Errorf("GetConfig() = %p, want the same pointer %p", got, cfg)
	}
}

func TestSaveConfig(t *testing.T) {
	core.SetDataDir(t.TempDir())

	// A real, but not "available", sldl path so ensureSldlExecutable succeeds.
	sldlPath := filepath.Join(t.TempDir(), "sldl")
	if err := os.WriteFile(sldlPath, []byte("fake"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	a := &App{sourceManager: core.NewSourceManager()}
	cfg := core.Config{
		Theme:              "light",
		SoulseekBinaryPath: sldlPath,
		SoulseekEnabled:    false,
		SourceOrder:        []string{"tidal"},
	}

	if err := a.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	if a.config == nil || a.config.Theme != "light" {
		t.Errorf("SaveConfig() did not update a.config, got %+v", a.config)
	}

	// Verify it actually persisted to disk (real behavior, isolated temp dir).
	data, err := os.ReadFile(core.GetConfigPath())
	if err != nil {
		t.Fatalf("reading saved config: %v", err)
	}
	if len(data) == 0 {
		t.Error("SaveConfig() wrote an empty file")
	}
}

func TestSaveConfig_AmazonEndpointPriority(t *testing.T) {
	core.SetDataDir(t.TempDir())
	sldlPath := filepath.Join(t.TempDir(), "sldl")
	if err := os.WriteFile(sldlPath, []byte("fake"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	a := &App{
		sourceManager: core.NewSourceManager(),
		amazonSource:  core.NewAmazonSource(),
	}
	baseCfg := core.Config{SoulseekBinaryPath: sldlPath}

	// Priority endpoint set, no override -> self-host prepended before the public pool.
	cfg := baseCfg
	cfg.AmazonPriorityEndpoints = []string{"https://my-amazon-proxy.example"}
	if err := a.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	snap := a.amazonSource.PoolSnapshot()
	if len(snap) == 0 || snap[0].URL != "https://my-amazon-proxy.example" {
		t.Fatalf("priority endpoint not prepended, got %+v", snap)
	}

	// Override set -> total override, priority field ignored.
	cfg = baseCfg
	cfg.AmazonPriorityEndpoints = []string{"https://my-amazon-proxy.example"}
	cfg.AmazonProxyEndpoints = []string{"https://override.example"}
	if err := a.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	snap = a.amazonSource.PoolSnapshot()
	if len(snap) != 1 || snap[0].URL != "https://override.example" {
		t.Fatalf("override endpoints not applied exclusively, got %+v", snap)
	}
}

func TestSetSourceOrder(t *testing.T) {
	tests := []struct {
		name    string
		order   []string
		wantErr string
	}{
		{name: "empty order", order: nil, wantErr: "source order cannot be empty"},
		{name: "unknown source", order: []string{"napster"}, wantErr: "unknown source: napster"},
		{name: "duplicate source", order: []string{"tidal", "tidal"}, wantErr: "duplicate source: tidal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &App{}
			err := a.SetSourceOrder(tt.order)
			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("SetSourceOrder(%v) error = %v, want %q", tt.order, err, tt.wantErr)
			}
		})
	}

	t.Run("valid order persists", func(t *testing.T) {
		core.SetDataDir(t.TempDir())
		a := &App{config: &core.Config{}}
		if err := a.SetSourceOrder([]string{"tidal", "qobuz"}); err != nil {
			t.Fatalf("SetSourceOrder() error = %v", err)
		}
		if got := a.config.SourceOrder; len(got) != 2 || got[0] != "tidal" || got[1] != "qobuz" {
			t.Errorf("SetSourceOrder() did not update a.config.SourceOrder, got %v", got)
		}
	})
}

func TestResetToDefaults(t *testing.T) {
	core.SetDataDir(t.TempDir())

	// a.logBuffer left nil so the runtime.EventsEmit branch (which requires a
	// real Wails runtime context and would otherwise call log.Fatalf) is skipped.
	a := &App{config: &core.Config{DownloadFolder: "/music/keep-me"}}

	cfg, err := a.ResetToDefaults()
	if err != nil {
		t.Fatalf("ResetToDefaults() error = %v", err)
	}
	if cfg.DownloadFolder != "/music/keep-me" {
		t.Errorf("ResetToDefaults() DownloadFolder = %q, want preserved %q", cfg.DownloadFolder, "/music/keep-me")
	}
	if a.config != cfg {
		t.Error("ResetToDefaults() did not update a.config to the new default config")
	}
}

func TestGetConnectionStatus(t *testing.T) {
	t.Run("nil spotifySearch", func(t *testing.T) {
		a := &App{}
		got := a.GetConnectionStatus()
		if got["tidalReady"] != true || got["spotifySearch"] != false {
			t.Errorf("GetConnectionStatus() = %v", got)
		}
	})
	t.Run("non-nil spotifySearch", func(t *testing.T) {
		a := &App{spotifySearch: &core.SpotifyClient{}}
		got := a.GetConnectionStatus()
		if got["spotifySearch"] != true {
			t.Errorf("GetConnectionStatus() = %v, want spotifySearch=true", got)
		}
	})
}

func TestCheckAPIStatus(t *testing.T) {
	t.Run("all nil", func(t *testing.T) {
		a := &App{}
		if got := a.CheckAPIStatus(); got != nil {
			t.Errorf("CheckAPIStatus() = %v, want nil", got)
		}
	})
	t.Run("downloader present", func(t *testing.T) {
		a := &App{downloader: core.NewTidalHifiService()}
		got := a.CheckAPIStatus()
		if len(got) == 0 {
			t.Fatal("CheckAPIStatus() = empty, want at least one endpoint from the default pool")
		}
		for _, s := range got {
			if s.Status != "online" && s.Status != "offline" && s.Status != "slow" {
				t.Errorf("CheckAPIStatus() entry has unexpected Status %q", s.Status)
			}
		}
	})
}

func TestEndpointStatToStatus(t *testing.T) {
	tests := []struct {
		state      string
		wantStatus string
	}{
		{"dead", "offline"},
		{"blacklisted", "slow"},
		{"probation", "slow"},
		{"live", "online"},
		{"", "online"}, // default branch
	}
	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			ep := core.EndpointStat{URL: "https://example.com/api", State: tt.state, LatencyMs: 42}
			got := endpointStatToStatus("Tidal HiFi", ep)
			if got.Status != tt.wantStatus {
				t.Errorf("endpointStatToStatus(state=%q).Status = %q, want %q", tt.state, got.Status, tt.wantStatus)
			}
			if got.Name != "Tidal HiFi — example.com/api" {
				t.Errorf("endpointStatToStatus() Name = %q, want host stripped of scheme", got.Name)
			}
			if got.LatencyMs != 42 {
				t.Errorf("endpointStatToStatus() LatencyMs = %d, want 42", got.LatencyMs)
			}
		})
	}
}
