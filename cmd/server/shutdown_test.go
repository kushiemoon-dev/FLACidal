package main

import (
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestShutdownAll(t *testing.T) {
	newListener := func(t *testing.T) net.Listener {
		t.Helper()
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("test setup failed: %v", err)
		}
		t.Cleanup(func() { ln.Close() })
		return ln
	}

	t.Run("everything stops: database closed after the downloads", func(t *testing.T) {
		var order []string
		ln := newListener(t)
		shutdownAll(
			func() { order = append(order, "downloads") },
			func() { order = append(order, "db") },
			func() error { order = append(order, "http"); return nil },
			ln, time.Second,
		)
		want := []string{"http", "downloads", "db"}
		if len(order) != len(want) {
			t.Fatalf("order = %v, want %v", order, want)
		}
		for i := range want {
			if order[i] != want[i] {
				t.Fatalf("order = %v, want %v", order, want)
			}
		}
		if _, err := ln.Accept(); err == nil {
			t.Error("listener still open after shutdownAll")
		}
	})

	t.Run("a download that never ends does not hold the shutdown", func(t *testing.T) {
		block := make(chan struct{})
		defer close(block)
		var dbClosed atomic.Bool
		start := time.Now()
		shutdownAll(
			func() { <-block },
			func() { dbClosed.Store(true) },
			func() error { return nil },
			newListener(t), 200*time.Millisecond,
		)
		if d := time.Since(start); d > 2*time.Second {
			t.Errorf("shutdownAll took %s, want it bounded by the timeout", d)
		}
		if dbClosed.Load() {
			t.Error("database closed while a download worker may still write to it")
		}
	})
}
