package app

import (
	"sync"
	"testing"
	"time"

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

type logEvents struct {
	mu     sync.Mutex
	events []core.LogEntry
}

func (l *logEvents) emit(name string, data any) {
	if name != "log" {
		return
	}
	l.mu.Lock()
	l.events = append(l.events, data.(core.LogEntry))
	l.mu.Unlock()
}

func (l *logEvents) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.events)
}

func waitEvents(t *testing.T, l *logEvents, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && l.count() < want {
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond) // let a stray duplicate show up
	if got := l.count(); got != want {
		t.Fatalf("log events = %d, want %d", got, want)
	}
}

func TestWireLogEvents_BufferEntryEmitsOneEvent(t *testing.T) {
	ev := &logEvents{}
	a := &App{logBuffer: core.NewLogBuffer(10), emit: ev.emit}
	a.wireLogEvents()
	defer a.cancelLogs()

	a.logBuffer.Info("from downloader")
	waitEvents(t, ev, 1)
	if ev.events[0].Message != "from downloader" {
		t.Errorf("event = %+v", ev.events[0])
	}
}

func TestAddLog_EmitsExactlyOneEvent(t *testing.T) {
	ev := &logEvents{}
	a := &App{logBuffer: core.NewLogBuffer(10), emit: ev.emit}
	a.wireLogEvents()
	defer a.cancelLogs()

	a.AddLog("info", "hello")
	waitEvents(t, ev, 1)
}
