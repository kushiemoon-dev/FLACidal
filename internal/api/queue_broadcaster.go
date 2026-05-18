package api

import (
	"sync"

	"github.com/google/uuid"
)

// QueueEvent is a typed event emitted by the download system.
type QueueEvent struct {
	Type     string     `json:"type"` // "queued"|"started"|"progress"|"completed"|"failed"|"snapshot"
	JobID    string     `json:"jobId"`
	Title    string     `json:"title,omitempty"`
	Artist   string     `json:"artist,omitempty"`
	Progress int        `json:"progress,omitempty"` // 0–100
	Error    string     `json:"error,omitempty"`
	Jobs     []QueueJob `json:"jobs,omitempty"` // populated for "snapshot"
}

// QueueJob is a lightweight job summary sent in snapshots.
type QueueJob struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
}

// QueueBroadcaster fans out QueueEvents to all current WebSocket subscribers.
type QueueBroadcaster struct {
	subs    sync.Map // id(string) → chan QueueEvent
	snapsMu sync.RWMutex
	snaps   map[string]QueueJob // id → latest known state
}

// NewQueueBroadcaster creates a ready-to-use broadcaster.
func NewQueueBroadcaster() *QueueBroadcaster {
	return &QueueBroadcaster{
		snaps: make(map[string]QueueJob),
	}
}

// Subscribe registers a new subscriber and returns its id and receive channel.
func (b *QueueBroadcaster) Subscribe() (id string, ch <-chan QueueEvent) {
	id = uuid.New().String()
	c := make(chan QueueEvent, 64)
	b.subs.Store(id, c)
	return id, c
}

// Unsubscribe removes a subscriber and closes its channel.
func (b *QueueBroadcaster) Unsubscribe(id string) {
	if v, ok := b.subs.LoadAndDelete(id); ok {
		close(v.(chan QueueEvent))
	}
}

// Broadcast sends an event to every subscriber (non-blocking; drops if channel full).
// It also maintains the internal snapshot state.
func (b *QueueBroadcaster) Broadcast(event QueueEvent) {
	b.updateSnapshot(event)

	b.subs.Range(func(_, v interface{}) bool {
		ch := v.(chan QueueEvent)
		select {
		case ch <- event:
		default:
			// subscriber too slow — drop
		}
		return true
	})
}

// Snapshot returns the current known state of all jobs.
func (b *QueueBroadcaster) Snapshot() []QueueJob {
	b.snapsMu.RLock()
	defer b.snapsMu.RUnlock()
	jobs := make([]QueueJob, 0, len(b.snaps))
	for _, j := range b.snaps {
		jobs = append(jobs, j)
	}
	return jobs
}

func (b *QueueBroadcaster) updateSnapshot(event QueueEvent) {
	b.snapsMu.Lock()
	defer b.snapsMu.Unlock()

	switch event.Type {
	case "completed", "failed":
		// Remove finished jobs from snapshot so they don't appear in new connections
		delete(b.snaps, event.JobID)
	case "queued", "started", "progress":
		status := event.Type
		if status == "started" {
			status = "downloading"
		}
		existing, ok := b.snaps[event.JobID]
		if ok {
			existing.Status = status
			existing.Progress = event.Progress
			b.snaps[event.JobID] = existing
		} else {
			b.snaps[event.JobID] = QueueJob{
				ID:       event.JobID,
				Title:    event.Title,
				Artist:   event.Artist,
				Status:   status,
				Progress: event.Progress,
			}
		}
	}
}
