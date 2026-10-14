package api

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func testSchedule() probeSchedule {
	return probeSchedule{initial: 5 * time.Millisecond, interval: 10 * time.Millisecond, jitter: 0.5, rand: func() float64 { return 0.25 }}
}

func TestRunProbeLoop_RepeatsThenStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	done := make(chan struct{})
	go func() {
		runProbeLoop(ctx, testSchedule(), func(context.Context) { calls.Add(1) })
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	for calls.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("only %d probe calls", calls.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("loop did not stop after cancel")
	}
}

func TestRunProbeLoop_CancelDuringInitialWaitMakesNoCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	sc := testSchedule()
	sc.initial = time.Hour
	done := make(chan struct{})
	go func() {
		runProbeLoop(ctx, sc, func(context.Context) { calls.Add(1) })
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("loop did not stop during the initial wait")
	}
	if calls.Load() != 0 {
		t.Errorf("calls = %d, want 0", calls.Load())
	}
}
