package app

import (
	"os"
	"path/filepath"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Deliberately left uncovered here:
//   - OpenConfigFolder / openFolder: these spawn a real OS file-manager process
//     (xdg-open/open/explorer), which isn't safe to trigger from a test.

func TestGetConfig(t *testing.T) {
	cfg := &core.Config{Theme: "dark"}
	a := &App{config: cfg}
	if got := a.GetConfig(); got != cfg {
		t.Errorf("GetConfig() = %p, want the same pointer %p", got, cfg)
	}
}

func TestSaveConfig(t *testing.T) {
	core.SetDataDir(t.TempDir())

	// A genuine, if not "available", sldl path so EnsureSldlExecutable doesn't error.
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
		// SaveConfig now logs through applyPriorityEndpoints whenever priority
		// endpoints are configured (see Startup), so a logBuffer is required here
		// the same way Startup always has one before SaveConfig can ever run.
		logBuffer: core.NewLogBuffer(500),
	}
	baseCfg := core.Config{SoulseekBinaryPath: sldlPath}

	// With only a priority endpoint set (no override), it's promoted to tier1 via
	// SetPriorityEndpoints, which the pool always places ahead of the base (tier2)
	// pool set separately via SetEndpoints — see Startup for the equivalent wiring.
	cfg := baseCfg
	cfg.AmazonPriorityEndpoints = []string{"https://my-amazon-proxy.example"}
	if err := a.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	snap := a.amazonSource.PoolSnapshot()
	if len(snap) == 0 || snap[0].URL != "https://my-amazon-proxy.example" {
		t.Fatalf("priority endpoint not prepended, got %+v", snap)
	}

	// An override replaces only the base (tier2) pool; the priority endpoint is
	// still layered on top via SetPriorityEndpoints regardless of the override,
	// same as Startup — so both entries are present, with priority still first.
	cfg = baseCfg
	cfg.AmazonPriorityEndpoints = []string{"https://my-amazon-proxy.example"}
	cfg.AmazonProxyEndpoints = []string{"https://override.example"}
	if err := a.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	snap = a.amazonSource.PoolSnapshot()
	if len(snap) != 2 || snap[0].URL != "https://my-amazon-proxy.example" || snap[1].URL != "https://override.example" {
		t.Fatalf("expected priority endpoint layered ahead of the override, got %+v", snap)
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
		if got := a.config.SourceOrder; len(got) != 3 || got[0] != "soulseek" || got[1] != "tidal" || got[2] != "qobuz" {
			t.Errorf("SetSourceOrder() did not update a.config.SourceOrder, got %v", got)
		}
	})
}

func TestValidateSourceOrder(t *testing.T) {
	tests := []struct {
		name    string
		order   []string
		want    []string
		wantErr string
	}{
		{name: "empty order", order: nil, wantErr: "source order cannot be empty"},
		{name: "unknown source", order: []string{"napster"}, wantErr: "unknown source: napster"},
		{name: "duplicate source", order: []string{"tidal", "tidal"}, wantErr: "duplicate source: tidal"},
		{name: "omits soulseek: prepended", order: []string{"tidal", "qobuz"}, want: []string{"soulseek", "tidal", "qobuz"}},
		{name: "contains soulseek not first: unchanged", order: []string{"tidal", "soulseek", "qobuz"}, want: []string{"tidal", "soulseek", "qobuz"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateSourceOrder(tt.order)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Errorf("ValidateSourceOrder(%v) error = %v, want %q", tt.order, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateSourceOrder(%v) unexpected error = %v", tt.order, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ValidateSourceOrder(%v) = %v, want %v", tt.order, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ValidateSourceOrder(%v) = %v, want %v", tt.order, got, tt.want)
				}
			}
		})
	}
}

func TestResolveAndPersistSourceOrder(t *testing.T) {
	t.Run("already set: returned unchanged, not persisted", func(t *testing.T) {
		core.SetDataDir(t.TempDir())
		config := &core.Config{SourceOrder: []string{"tidal"}}
		got := resolveAndPersistSourceOrder(config, nil)
		if len(got) != 1 || got[0] != "tidal" {
			t.Errorf("resolveAndPersistSourceOrder() = %v, want [tidal]", got)
		}
		if len(config.SourceOrder) != 1 || config.SourceOrder[0] != "tidal" {
			t.Errorf("config.SourceOrder = %v, want unchanged [tidal]", config.SourceOrder)
		}
	})

	t.Run("empty: resolved via DefaultSourceOrder and persisted", func(t *testing.T) {
		core.SetDataDir(t.TempDir())
		config := &core.Config{}
		want := core.DefaultSourceOrder(config)

		got := resolveAndPersistSourceOrder(config, nil)

		if len(got) != len(want) {
			t.Fatalf("resolveAndPersistSourceOrder() = %v, want %v", got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("resolveAndPersistSourceOrder() = %v, want %v", got, want)
			}
		}
		if len(config.SourceOrder) != len(want) {
			t.Errorf("config.SourceOrder = %v, want mutated to match %v", config.SourceOrder, want)
		}

		data, err := os.ReadFile(core.GetConfigPath())
		if err != nil {
			t.Fatalf("reading persisted config: %v", err)
		}
		if len(data) == 0 {
			t.Error("resolveAndPersistSourceOrder() did not persist a non-empty config file")
		}
	})
}

func TestResetToDefaults(t *testing.T) {
	core.SetDataDir(t.TempDir())

	// a.logBuffer is left nil so we skip the runtime.EventsEmit branch, which
	// needs an actual Wails runtime context and would call log.Fatalf otherwise.
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
		{"", "online"}, // falls through to the default case
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
