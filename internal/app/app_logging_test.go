package app

import (
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Characterization tests for the "Logging Methods" section of app.go.
//
// NOT tested here (documented, not fixed):
//   - AddLog's non-nil-logBuffer branch calls runtime.EventsEmit(a.ctx, "log",
//     entry), which requires a real Wails runtime context and otherwise calls
//     log.Fatalf (see wails/v2/pkg/runtime/runtime.go) — that would abort the
//     whole test process. Only the nil-logBuffer no-op branch is exercised.

func TestGetLogs(t *testing.T) {
	t.Run("nil logBuffer", func(t *testing.T) {
		a := &App{}
		if got := a.GetLogs(); len(got) != 0 {
			t.Errorf("GetLogs() with nil logBuffer = %v, want empty", got)
		}
	})
	t.Run("real logBuffer", func(t *testing.T) {
		lb := core.NewLogBuffer(10)
		lb.Info("hello")
		a := &App{logBuffer: lb}
		got := a.GetLogs()
		if len(got) != 1 || got[0].Message != "hello" {
			t.Errorf("GetLogs() = %v, want one entry with Message=hello", got)
		}
	})
}

func TestClearLogs(t *testing.T) {
	t.Run("nil logBuffer is a no-op", func(t *testing.T) {
		a := &App{}
		a.ClearLogs() // must not panic
	})
	t.Run("real logBuffer", func(t *testing.T) {
		lb := core.NewLogBuffer(10)
		lb.Info("hello")
		a := &App{logBuffer: lb}
		a.ClearLogs()
		if got := a.GetLogs(); len(got) != 0 {
			t.Errorf("GetLogs() after ClearLogs() = %v, want empty", got)
		}
	})
}

func TestAddLog_NilLogBufferNoOp(t *testing.T) {
	a := &App{}
	a.AddLog("info", "hello") // must not panic, must not touch a.ctx
}
