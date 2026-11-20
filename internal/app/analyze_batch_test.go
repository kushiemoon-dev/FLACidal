package app

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBatchRegistry(t *testing.T) {
	var r BatchRegistry
	if _, _, err := r.Begin(""); err == nil {
		t.Error("empty id must be refused")
	}
	ctx, done, err := r.Begin("b1")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.Begin("b1"); err == nil {
		t.Error("duplicate active id must be refused")
	}
	if r.Cancel("unknown") {
		t.Error("Cancel of an unknown id must be false")
	}
	if !r.Cancel("b1") {
		t.Error("Cancel of an active id must be true")
	}
	if ctx.Err() == nil {
		t.Error("context should be cancelled")
	}
	done()
	if r.Cancel("b1") {
		t.Error("id must be gone after done")
	}
	if _, _, err := r.Begin("b1"); err != nil {
		t.Errorf("id should be reusable after done: %v", err)
	}
}

func TestAnalyzeBatch_Guards(t *testing.T) {
	a := &App{}
	if _, err := a.AnalyzeBatch("", []string{"/x.flac"}); err == nil {
		t.Error("empty batch id must be refused")
	}
	if _, err := a.AnalyzeBatch("b", nil); err == nil {
		t.Error("empty path list must be refused")
	}
	paths := make([]string, 5001)
	for i := range paths {
		paths[i] = fmt.Sprintf("/x/%d.flac", i)
	}
	_, err := a.AnalyzeBatch("b", paths)
	if err == nil || !strings.Contains(err.Error(), "too many files: 5001 (maximum 5000)") {
		t.Errorf("err = %v", err)
	}
	if a.CancelAnalyzeBatch("b") {
		t.Error("nothing should be registered after a refused batch")
	}
}

// Progress must go through the serial emitter (one ExecuteJS at a time, in
// order, none merged), not straight to the Wails runtime.
func TestAnalyzeBatch_ProgressUsesSerialEmitter(t *testing.T) {
	var mu sync.Mutex
	var got []map[string]any
	a := &App{events: newSerialEmitter(func(name string, data any) {
		if name != "analyze-progress" {
			return
		}
		mu.Lock()
		got = append(got, data.(map[string]any))
		mu.Unlock()
	}, 0)}

	paths := []string{"/nonexistent/a.flac", "/nonexistent/b.flac", "/nonexistent/c.flac"}
	results, err := a.AnalyzeBatch("b1", paths)
	if err != nil || len(results) != 3 {
		t.Fatalf("results=%d err=%v", len(results), err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n == 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 3 {
		t.Fatalf("%d progress events, wanted 3 (none may be dropped or merged)", len(got))
	}
	for i, e := range got {
		if e["batchId"] != "b1" || e["done"] != i+1 || e["total"] != 3 {
			t.Errorf("event %d = %v", i, e)
		}
	}
}
