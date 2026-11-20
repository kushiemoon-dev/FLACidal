package app

import (
	"fmt"

	core "github.com/kushiemoon-dev/flacidal-core"
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
		emit = a.emitEvent
	}
	a.cancelLogs = a.logBuffer.Subscribe(func(e core.LogEntry) { emit("log", e) })
}

// logDownloading writes the "Now downloading" line once per track: the core
// reports progress several times a second, and one log line per tick would
// flood the log and the event queue.
func (a *App) logDownloading(trackID int) {
	if _, seen := a.downloadingLogged.LoadOrStore(trackID, struct{}{}); !seen {
		a.logBuffer.Info(fmt.Sprintf("Now downloading track %d...", trackID))
	}
}

// clearDownloading lets a retried track log its "Now downloading" line again.
func (a *App) clearDownloading(trackID int) {
	a.downloadingLogged.Delete(trackID)
}

func (a *App) GetLogFilePath() string {
	return core.GetLogFilePath()
}
