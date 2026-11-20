package app

import (
	"fmt"
	"strings"
	"testing"
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
