package app

import (
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const serialEmitterBuffer = 256

type serialEvent struct {
	name string
	data any
	key  string // set for a coalescible event, see SendCoalesced
}

// serialEmitter funnels every front end event through one goroutine, so
// ExecuteJS calls never overlap, which otherwise crashes WebKit on Linux.
type serialEmitter struct {
	mu       sync.Mutex
	notEmpty *sync.Cond
	notFull  *sync.Cond
	queue    []*serialEvent
	pending  map[string]*serialEvent // coalescible events still waiting in queue
	closed   bool
}

// newSerialEmitter starts the goroutine; pause is waited after each event so
// WebKit/GTK can process the JS.
func newSerialEmitter(emit func(name string, data any), pause time.Duration) *serialEmitter {
	s := &serialEmitter{pending: map[string]*serialEvent{}}
	s.notEmpty = sync.NewCond(&s.mu)
	s.notFull = sync.NewCond(&s.mu)
	go func() {
		for {
			s.mu.Lock()
			for len(s.queue) == 0 && !s.closed {
				s.notEmpty.Wait()
			}
			if s.closed {
				s.mu.Unlock()
				return
			}
			ev := s.queue[0]
			s.queue = s.queue[1:]
			if ev.key != "" && s.pending[ev.key] == ev {
				delete(s.pending, ev.key)
			}
			s.notFull.Signal()
			s.mu.Unlock()

			emit(ev.name, ev.data)
			time.Sleep(pause)
		}
	}()
	return s
}

// Send blocks briefly if the buffer is full, trivial next to download time.
// It is a barrier: a progress tick sent afterwards is never merged with one
// queued before it, so events keep their order.
func (s *serialEmitter) Send(name string, data any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.waitForRoom()
	if s.closed {
		return
	}
	clear(s.pending)
	s.push(&serialEvent{name: name, data: data})
}

// SendCoalesced is for events that only repeat the latest state (progress
// ticks): while one with the same key is still queued, it is replaced instead
// of queueing another, so a busy download cannot flood the queue.
func (s *serialEmitter) SendCoalesced(key, name string, data any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p := s.pending[key]; p != nil {
		p.name, p.data = name, data
		return
	}
	s.waitForRoom()
	if s.closed {
		return
	}
	ev := &serialEvent{name: name, data: data, key: key}
	s.pending[key] = ev
	s.push(ev)
}

// TrySend drops the event when the buffer is full. For log lines, which stay
// readable in the log buffer anyway.
func (s *serialEmitter) TrySend(name string, data any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || len(s.queue) >= serialEmitterBuffer {
		return
	}
	s.push(&serialEvent{name: name, data: data})
}

// Close drops the events still waiting and stops the goroutine once the event
// it may be sending has returned. Later sends are ignored, and a Send waiting
// for room is released. Without it the queue would keep calling a window that
// is being destroyed for as long as its backlog lasts. Safe to call twice.
func (s *serialEmitter) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.queue = nil
	clear(s.pending)
	s.notEmpty.Broadcast()
	s.notFull.Broadcast()
}

// waitForRoom must be called with s.mu held. On return the emitter may be closed.
func (s *serialEmitter) waitForRoom() {
	for len(s.queue) >= serialEmitterBuffer && !s.closed {
		s.notFull.Wait()
	}
}

// push must be called with s.mu held.
func (s *serialEmitter) push(ev *serialEvent) {
	s.queue = append(s.queue, ev)
	s.notEmpty.Signal()
}

// emitEvent is the only way for App methods to reach the front end. Before
// Startup has built the emitter (tests, headless) it falls back to Wails directly.
func (a *App) emitEvent(name string, data any) {
	if a.events != nil {
		a.events.Send(name, data)
		return
	}
	runtime.EventsEmit(a.ctx, name, data)
}

// emitCoalesced is emitEvent for repeated progress updates, see SendCoalesced.
func (a *App) emitCoalesced(key, name string, data any) {
	if a.events != nil {
		a.events.SendCoalesced(key, name, data)
		return
	}
	a.emitEvent(name, data)
}
