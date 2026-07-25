package main

import (
	"os"
	"path/filepath"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func TestRegisterSoulseekSource(t *testing.T) {
	t.Run("disabled: never registered", func(t *testing.T) {
		sm := core.NewSourceManager()
		registerSoulseekSource(sm, &core.Config{SoulseekEnabled: false})
		if _, ok := sm.GetSource("soulseek"); ok {
			t.Error("registerSoulseekSource() registered soulseek while disabled")
		}
	})

	t.Run("enabled but binary missing: not registered", func(t *testing.T) {
		sm := core.NewSourceManager()
		registerSoulseekSource(sm, &core.Config{
			SoulseekEnabled:    true,
			SoulseekUsername:   "user",
			SoulseekPassword:   "pass",
			SoulseekBinaryPath: filepath.Join(t.TempDir(), "does-not-exist"),
		})
		if _, ok := sm.GetSource("soulseek"); ok {
			t.Error("registerSoulseekSource() registered soulseek with a missing binary")
		}
	})

	t.Run("enabled, binary present, credentials set: registered", func(t *testing.T) {
		sldlPath := filepath.Join(t.TempDir(), "sldl")
		if err := os.WriteFile(sldlPath, []byte("fake"), 0755); err != nil {
			t.Fatalf("setup: %v", err)
		}
		sm := core.NewSourceManager()
		registerSoulseekSource(sm, &core.Config{
			SoulseekEnabled:    true,
			SoulseekUsername:   "user",
			SoulseekPassword:   "pass",
			SoulseekBinaryPath: sldlPath,
		})
		if _, ok := sm.GetSource("soulseek"); !ok {
			t.Error("registerSoulseekSource() did not register an available, enabled soulseek source")
		}
	})
}
