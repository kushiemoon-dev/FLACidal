package api

import (
	"sync"

	"github.com/google/uuid"
)

type QueueEvent struct {
	Type     string     `json:"type"` // "queued"|"started"|"progress"|"completed"|"failed"|"snapshot"
	JobID    string     `json:"jobId"`
	Title    string     `json:"title,omitempty"`
	Artist   string     `json:"artist,omitempty"`
	Progress int        `json:"progress,omitempty"` // percentage, 0 through 100
	Error    string     `json:"error,omitempty"`
	Jobs     []QueueJob `json:"jobs,omitempty"` // only set on a "snapshot" event
}

type QueueJob struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
}

type QueueBroadcaster struct {
	subs    sync.Map // subscriber id -> chan QueueEvent
	snapsMu sync.RWMutex
	snaps   map[string]QueueJob
}

func NewQueueBroadcaster() *QueueBroadcaster {
	return &QueueBroadcaster{
		snaps: make(map[string]QueueJob),
	}
}

func (b *QueueBroadcaster) Subscribe() (id string, ch <-chan QueueEvent) {
	id = uuid.New().String()
	c := make(chan QueueEvent, 64)
	b.subs.Store(id, c)
	return id, c
}

func (b *QueueBroadcaster) Unsubscribe(id string) {
	if v, ok := b.subs.LoadAndDelete(id); ok {
		close(v.(chan QueueEvent))
	}
}

// Broadcast pushes event to every subscriber without blocking, dropping it for
// any subscriber whose channel is currently full.
func (b *QueueBroadcaster) Broadcast(event QueueEvent) {
	b.updateSnapshot(event)

	b.subs.Range(func(_, v interface{}) bool {
		ch := v.(chan QueueEvent)
		select {
		case ch <- event:
		default:
		}
		return true
	})
}

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
		// Drop finished jobs so newly connecting clients don't see them in the snapshot
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
