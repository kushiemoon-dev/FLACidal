package backend

import (
	"sync"
	"time"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"` // "info", "warn", "error", "success"
	Message   string `json:"message"`
}

// LogBuffer stores log entries with a maximum size
type LogBuffer struct {
	entries []LogEntry
	maxSize int
	mu      sync.RWMutex
}

// NewLogBuffer creates a new log buffer with specified max size
func NewLogBuffer(maxSize int) *LogBuffer {
	return &LogBuffer{
		entries: make([]LogEntry, 0),
		maxSize: maxSize,
	}
}

// Add adds a new log entry
func (lb *LogBuffer) Add(level, message string) LogEntry {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now().Format("15:04:05"),
		Level:     level,
		Message:   message,
	}

	lb.entries = append(lb.entries, entry)

	// Trim if exceeds max size
	if len(lb.entries) > lb.maxSize {
		lb.entries = lb.entries[1:]
	}

	return entry
}

// Info adds an info log
func (lb *LogBuffer) Info(message string) LogEntry {
	return lb.Add("info", message)
}

// Success adds a success log
func (lb *LogBuffer) Success(message string) LogEntry {
	return lb.Add("success", message)
}

// Warn adds a warning log
func (lb *LogBuffer) Warn(message string) LogEntry {
	return lb.Add("warn", message)
}

// Error adds an error log
func (lb *LogBuffer) Error(message string) LogEntry {
	return lb.Add("error", message)
}

// GetAll returns all log entries
func (lb *LogBuffer) GetAll() []LogEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	result := make([]LogEntry, len(lb.entries))
	copy(result, lb.entries)
	return result
}

// Clear removes all log entries
func (lb *LogBuffer) Clear() {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.entries = make([]LogEntry, 0)
}

// Count returns the number of log entries
func (lb *LogBuffer) Count() int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return len(lb.entries)
}
