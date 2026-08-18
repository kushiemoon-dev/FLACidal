package app

import (
	core "github.com/kushiemoon-dev/flacidal-core"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) GetLogs() []core.LogEntry {
	if a.logBuffer == nil {
		return []core.LogEntry{}
	}
	return a.logBuffer.GetAll()
}

func (a *App) ClearLogs() {
	if a.logBuffer != nil {
		a.logBuffer.Clear()
	}
}

// Intended for testing and debugging.
func (a *App) AddLog(level, message string) {
	if a.logBuffer != nil {
		entry := a.logBuffer.Add(level, message)
		runtime.EventsEmit(a.ctx, "log", entry)
	}
}
