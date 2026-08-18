package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Behavioral snapshot tests covering app.go's package-level helpers and lifecycle hooks.
//
// Deliberately left uncovered here:
//   - startup(ctx): the full bootstrap sequence — it opens the real
//     config/database, makes a live network call via core.InitTidalEndpoints,
//     and starts goroutines bound to runtime.EventsEmit(ctx, ...). Without an
//     actual Wails runtime context, any runtime.* call panics via log.Fatalf
//     (see wails/v2/pkg/runtime/runtime.go). This is an integration-level
//     entrypoint rather than something unit-testable.
//   - shutdown(ctx): stops the download manager and persists config/db.
//     Aside from trivial nil-guards, testing it meaningfully needs the same
//     real dependencies startup does.

func TestDefaultSldlPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this path only applies to darwin/linux; the test runner isn't windows")
	}
	got := DefaultSldlPath()
	homeDir, _ := os.UserHomeDir()
	want := filepath.Join(homeDir, ".local", "share", "flacidal", "sldl")
	if got != want {
		t.Errorf("DefaultSldlPath() = %q, want %q", got, want)
	}
}

func TestEnsureSldlExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows has different chmod semantics; EnsureSldlExecutable is a no-op there")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "sldl")
	if err := os.WriteFile(path, []byte("fake binary"), 0644); err != nil {
		t.Fatalf("test setup failed: %v", err)
	}

	if err := EnsureSldlExecutable(path); err != nil {
		t.Fatalf("EnsureSldlExecutable() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after EnsureSldlExecutable: %v", err)
	}
	if info.Mode().Perm()&0100 == 0 {
		t.Errorf("EnsureSldlExecutable() did not set the owner-executable bit, mode = %v", info.Mode())
	}
}

func TestEnsureSldlExecutable_MissingFile(t *testing.T) {
	err := EnsureSldlExecutable(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Error("EnsureSldlExecutable() on a missing file: want error, got nil")
	}
}
