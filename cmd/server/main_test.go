package main

import (
	"os"
	"path/filepath"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func TestApplyPriorityEndpoints(t *testing.T) {
	t.Run("empty urls: setPriority is not called", func(t *testing.T) {
		called := false
		applyPriorityEndpoints("test", func(urls []string) int {
			called = true
			return len(urls)
		}, nil)
		if called {
			t.Error("applyPriorityEndpoints() should not call setPriority when urls is empty")
		}
	})

	t.Run("non-empty urls: setPriority is called with the same list", func(t *testing.T) {
		var got []string
		applyPriorityEndpoints("test", func(urls []string) int {
			got = urls
			return len(urls)
		}, []string{"https://a.example", "https://b.example"})
		if len(got) != 2 {
			t.Errorf("applyPriorityEndpoints() called setPriority with %d urls, want 2", len(got))
		}
	})
}

func TestRegisterSoulseekSource(t *testing.T) {
	t.Run("disabled: never registered", func(t *testing.T) {
		sm := core.NewSourceManager()
		registerSoulseekSource(sm, &core.Config{SoulseekEnabled: false})
		if _, ok := sm.GetSource("soulseek"); ok {
			t.Error("registerSoulseekSource() should not register soulseek when it's disabled")
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
			t.Error("registerSoulseekSource() should not register soulseek when the binary is missing")
		}
	})

	t.Run("enabled, binary present, credentials set: registered", func(t *testing.T) {
		sldlPath := filepath.Join(t.TempDir(), "sldl")
		if err := os.WriteFile(sldlPath, []byte("fake"), 0755); err != nil {
			t.Fatalf("test setup failed: %v", err)
		}
		sm := core.NewSourceManager()
		registerSoulseekSource(sm, &core.Config{
			SoulseekEnabled:    true,
			SoulseekUsername:   "user",
			SoulseekPassword:   "pass",
			SoulseekBinaryPath: sldlPath,
		})
		if _, ok := sm.GetSource("soulseek"); !ok {
			t.Error("registerSoulseekSource() should register soulseek when it's enabled and available")
		}
	})
}
