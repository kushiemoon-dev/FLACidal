package app

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func TestSerialEmitter_NeverEmitsConcurrently(t *testing.T) {
	var inFlight, maxInFlight int32
	var mu sync.Mutex
	var got []int
	em := newSerialEmitter(func(_ string, data any) {
		n := atomic.AddInt32(&inFlight, 1)
		if n > atomic.LoadInt32(&maxInFlight) {
			atomic.StoreInt32(&maxInFlight, n)
		}
		time.Sleep(time.Millisecond)
		mu.Lock()
		got = append(got, data.(int))
		mu.Unlock()
		atomic.AddInt32(&inFlight, -1)
	}, 0)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); em.Send("progress", i) }()
		go func() { defer wg.Done(); em.TrySend("log", i) }()
	}
	wg.Wait()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n == 40 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if m := atomic.LoadInt32(&maxInFlight); m != 1 {
		t.Errorf("max concurrent emits = %d, want 1", m)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 40 {
		t.Errorf("emitted %d events, want 40", len(got))
	}
}

func TestSerialEmitter_TrySendDropsWhenFull(t *testing.T) {
	block := make(chan struct{})
	em := newSerialEmitter(func(string, any) { <-block }, 0)
	defer close(block)

	done := make(chan struct{})
	go func() {
		for i := 0; i < serialEmitterBuffer*2; i++ {
			em.TrySend("log", i)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("TrySend blocked on a full queue")
	}
}

type emitted struct {
	name string
	data any
}

func capturingEmitter() (*serialEmitter, chan emitted) {
	ch := make(chan emitted, 16)
	return newSerialEmitter(func(name string, data any) { ch <- emitted{name, data} }, 0), ch
}

func expectEvent(t *testing.T, ch chan emitted, name string, data any) {
	t.Helper()
	select {
	case ev := <-ch:
		if ev.name != name || ev.data != data {
			t.Errorf("event = %+v, want %s %v", ev, name, data)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("no %q event emitted", name)
	}
}

func TestPauseAndResumeEmitThroughSerialEmitter(t *testing.T) {
	em, ch := capturingEmitter()
	a := &App{
		downloadManager: core.NewDownloadManager(core.NewTidalHifiService(), 1),
		logBuffer:       core.NewLogBuffer(10),
		events:          em,
	}
	if !a.PauseDownloads() {
		t.Fatal("PauseDownloads() = false")
	}
	expectEvent(t, ch, "queue-paused", true)
	if !a.ResumeDownloads() {
		t.Fatal("ResumeDownloads() = false")
	}
	expectEvent(t, ch, "queue-paused", false)
}

func TestEmitEventUsesSerialEmitter(t *testing.T) {
	em, ch := capturingEmitter()
	a := &App{events: em}
	a.emitEvent("ffmpeg-install-progress", 42)
	expectEvent(t, ch, "ffmpeg-install-progress", 42)
}

// Overlapping ExecuteJS calls crash WebKit on Linux, so the front end must only
// be reached through the serial emitter. This pins every direct call site.
func TestRuntimeEventsEmitOnlyInTheSerialEmitterPath(t *testing.T) {
	allowed := map[string]int{"app.go": 1, "serial_emitter.go": 1}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Count(string(src), "runtime.EventsEmit("); got != allowed[f] {
			t.Errorf("%s calls runtime.EventsEmit %d times, want %d: route it through a.emitEvent", f, got, allowed[f])
		}
	}
}

func TestLogDownloadingOnlyOncePerTrack(t *testing.T) {
	a := &App{logBuffer: core.NewLogBuffer(10)}
	for i := 0; i < 5; i++ {
		a.logDownloading(7)
	}
	a.logDownloading(8)
	if got := len(a.logBuffer.GetAll()); got != 2 {
		t.Errorf("log lines = %d, want 2 (one per track)", got)
	}
	a.clearDownloading(7)
	a.logDownloading(7)
	if got := len(a.logBuffer.GetAll()); got != 3 {
		t.Errorf("log lines after a retry = %d, want 3", got)
	}
}

func TestSerialEmitter_CoalescesProgressTicksBetweenBarriers(t *testing.T) {
	release := make(chan struct{})
	var mu sync.Mutex
	var got []string
	em := newSerialEmitter(func(name string, data any) {
		mu.Lock()
		got = append(got, name+":"+data.(string))
		first := len(got) == 1
		mu.Unlock()
		if first {
			<-release
		}
	}, 0)

	em.Send("first", "x") // taken by the emitter goroutine, which then blocks
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n == 1 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	for _, v := range []string{"1", "2", "3", "4", "5"} {
		em.SendCoalesced("track:1", "tick", v)
	}
	em.SendCoalesced("track:2", "tick", "a")
	em.SendCoalesced("track:2", "tick", "b")
	em.Send("done", "d") // not coalescible: a barrier
	em.SendCoalesced("track:1", "tick", "6")
	em.SendCoalesced("track:1", "tick", "7")
	close(release)

	want := []string{"first:x", "tick:5", "tick:b", "done:d", "tick:7"}
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n >= len(want) {
			break
		}
		time.Sleep(time.Millisecond)
	}
	time.Sleep(20 * time.Millisecond) // let a stray extra event show up
	mu.Lock()
	defer mu.Unlock()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("emitted %v, want %v", got, want)
	}
}

func TestSerialEmitter_SendBlocksWhenFullThenResumes(t *testing.T) {
	block := make(chan struct{})
	started := make(chan struct{}, 1)
	em := newSerialEmitter(func(string, any) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-block
	}, 0)

	em.Send("first", 0)
	<-started // the emitter goroutine is now stuck in emit, the queue is empty
	for i := 0; i < serialEmitterBuffer; i++ {
		em.Send("fill", i)
	}

	done := make(chan struct{})
	go func() {
		em.Send("extra", 1)
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("Send returned while the queue was full")
	case <-time.After(50 * time.Millisecond):
	}
	close(block)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Send still blocked after the consumer resumed")
	}
}

func TestEmitCoalescedUsesSerialEmitter(t *testing.T) {
	em, ch := capturingEmitter()
	a := &App{events: em}
	a.emitCoalesced("ffmpeg", "ffmpeg-install-progress", 7)
	expectEvent(t, ch, "ffmpeg-install-progress", 7)
}

func TestSerialEmitter_CloseDropsBacklogAndStops(t *testing.T) {
	release := make(chan struct{})
	var mu sync.Mutex
	var got []string
	em := newSerialEmitter(func(name string, _ any) {
		mu.Lock()
		got = append(got, name)
		first := len(got) == 1
		mu.Unlock()
		if first {
			<-release
		}
	}, 0)

	em.Send("first", nil)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n == 1 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	for i := 0; i < 10; i++ {
		em.Send("backlog", i)
	}
	em.Close()
	em.Close() // idempotent

	done := make(chan struct{})
	go func() {
		em.Send("late", nil)
		em.SendCoalesced("k", "late", nil)
		em.TrySend("late", nil)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sending after Close blocked")
	}
	close(release)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !reflect.DeepEqual(got, []string{"first"}) {
		t.Errorf("emitted %v after Close, want only the in-flight event", got)
	}
}

func TestSerialEmitter_CloseReleasesBlockedSend(t *testing.T) {
	block := make(chan struct{})
	started := make(chan struct{}, 1)
	em := newSerialEmitter(func(string, any) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-block
	}, 0)
	defer close(block)

	em.Send("first", 0)
	<-started
	for i := 0; i < serialEmitterBuffer; i++ {
		em.Send("fill", i)
	}
	done := make(chan struct{})
	go func() {
		em.Send("extra", 1)
		close(done)
	}()
	time.Sleep(50 * time.Millisecond)
	em.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a Send waiting for room was not released by Close")
	}
}

func TestShutdownStopsTheEventQueue(t *testing.T) {
	em, ch := capturingEmitter()
	a := &App{events: em}
	a.Shutdown(context.Background())
	a.emitEvent("late", 1)
	select {
	case ev := <-ch:
		t.Errorf("event %+v emitted after Shutdown", ev)
	case <-time.After(50 * time.Millisecond):
	}
}
