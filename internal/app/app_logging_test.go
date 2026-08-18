package app

import (
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Behavioral snapshot tests covering the logging methods section of app.go.
//
// Deliberately left uncovered here:
//   - AddLog's non-nil-logBuffer branch invokes runtime.EventsEmit(a.ctx, "log",
//     entry), which needs an actual Wails runtime context and otherwise triggers
//     log.Fatalf (see wails/v2/pkg/runtime/runtime.go), aborting the whole test
//     process. Only the nil-logBuffer no-op branch runs here.

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
		a.ClearLogs()
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
	a.AddLog("info", "hello")
}
