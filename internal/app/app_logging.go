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
		a.logBuffer.Add(level, message)
	}
}

// wireLogEvents forwards every log buffer entry to the front end as a "log"
// event, so entries written by the downloader and the queue show up live too.
// This is the only place that emits "log".
func (a *App) wireLogEvents() {
	emit := a.emit
	if emit == nil {
		emit = func(event string, data any) { runtime.EventsEmit(a.ctx, event, data) }
	}
	a.cancelLogs = a.logBuffer.Subscribe(func(e core.LogEntry) { emit("log", e) })
}

func (a *App) GetLogFilePath() string {
	return core.GetLogFilePath()
}
