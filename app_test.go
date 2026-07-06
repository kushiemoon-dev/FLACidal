package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Characterization tests for app.go's package-level helpers and lifecycle hooks.
//
// NOT tested here (documented, not fixed):
//   - startup(ctx): full bootstrap — opens the real config/database, makes a live
//     network call via core.InitTidalEndpoints, and starts goroutines bound to
//     runtime.EventsEmit(ctx, ...). Without a real Wails runtime context, any
//     runtime.* call panics via log.Fatalf (see wails/v2/pkg/runtime/runtime.go).
//     This is an integration-level entrypoint, not a unit-testable method.
//   - shutdown(ctx): stops the download manager and persists config/db — trivial
//     nil-guards aside, exercising it meaningfully requires the same real
//     dependencies as startup.

func TestDefaultSldlPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("darwin/linux path only; current test runner is not windows")
	}
	got := defaultSldlPath()
	homeDir, _ := os.UserHomeDir()
	want := filepath.Join(homeDir, ".local", "share", "flacidal", "sldl")
	if got != want {
		t.Errorf("defaultSldlPath() = %q, want %q", got, want)
	}
}

func TestEnsureSldlExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod semantics differ on windows; ensureSldlExecutable is a no-op there")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "sldl")
	if err := os.WriteFile(path, []byte("fake binary"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := ensureSldlExecutable(path); err != nil {
		t.Fatalf("ensureSldlExecutable() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after ensureSldlExecutable: %v", err)
	}
	if info.Mode().Perm()&0100 == 0 {
		t.Errorf("ensureSldlExecutable() did not set the owner-executable bit, mode = %v", info.Mode())
	}
}

func TestEnsureSldlExecutable_MissingFile(t *testing.T) {
	err := ensureSldlExecutable(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Error("ensureSldlExecutable() on a missing file: want error, got nil")
	}
}
