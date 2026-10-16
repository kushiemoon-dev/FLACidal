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
//   - ExportConfig / ImportConfig: pop a real native save/open dialog through
//     runtime.SaveFileDialog / runtime.OpenFileDialog, which needs an actual
//     Wails runtime context, same as ExportFailedDownloads in app_queue_test.go.

func TestGetConfig(t *testing.T) {
	cfg := &core.Config{Theme: "dark"}
	a := &App{config: cfg}
	// GetConfig hands out a copy so callers cannot race with SaveConfig.
	if got := a.GetConfig(); got == cfg || got.Theme != "dark" {
		t.Errorf("GetConfig() = %+v, want a copy of %+v", got, cfg)
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
		if got := a.config.SourceOrder; len(got) != 3 || got[0] != "tidal" || got[1] != "qobuz" || got[2] != "soulseek" {
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
		{name: "omits soulseek: appended last", order: []string{"tidal", "qobuz"}, want: []string{"tidal", "qobuz", "soulseek"}},
		{name: "soulseek first: unchanged", order: []string{"soulseek", "tidal"}, want: []string{"soulseek", "tidal"}},
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
