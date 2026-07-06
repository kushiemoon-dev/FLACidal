package main

import (
	core "github.com/kushiemoon-dev/flacidal-core"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// =============================================================================
// Logging Methods (exposed to frontend)
// =============================================================================

// GetLogs returns all log entries
func (a *App) GetLogs() []core.LogEntry {
	if a.logBuffer == nil {
		return []core.LogEntry{}
	}
	return a.logBuffer.GetAll()
}

// ClearLogs clears all log entries
func (a *App) ClearLogs() {
	if a.logBuffer != nil {
		a.logBuffer.Clear()
	}
}

// AddLog adds a log entry (for testing/debug)
func (a *App) AddLog(level, message string) {
	if a.logBuffer != nil {
		entry := a.logBuffer.Add(level, message)
		// Emit log event to frontend
		runtime.EventsEmit(a.ctx, "log", entry)
	}
}
