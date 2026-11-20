package app

import (
	"context"
	"fmt"
	"os"
	"sync"

	core "github.com/kushiemoon-dev/flacidal-core"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// BatchRegistry tracks running analysis batches so they can be cancelled by id.
// It is shared by the Wails App and the HTTP server.
type BatchRegistry struct {
	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

// Begin registers id and returns its context and a release func that unregisters
// the id and cancels the context.
func (r *BatchRegistry) Begin(id string) (context.Context, func(), error) {
	if id == "" {
		return nil, nil, fmt.Errorf("a batch id is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, active := r.cancels[id]; active {
		return nil, nil, fmt.Errorf("batch %q is already running", id)
	}
	if r.cancels == nil {
		r.cancels = make(map[string]context.CancelFunc)
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancels[id] = cancel
	return ctx, func() {
		r.mu.Lock()
		delete(r.cancels, id)
		r.mu.Unlock()
		cancel()
	}, nil
}

// Cancel cancels the batch id and reports whether it was running.
func (r *BatchRegistry) Cancel(id string) bool {
	r.mu.Lock()
	cancel, ok := r.cancels[id]
	r.mu.Unlock()
	if ok {
		cancel()
	}
	return ok
}

// CheckBatchSize returns the shared refusal for an empty or oversized batch.
func CheckBatchSize(n int) error {
	if n == 0 {
		return fmt.Errorf("no files to analyze")
	}
	if n > core.MaxAnalyzeBatch {
		return fmt.Errorf("too many files: %d (maximum %d)", n, core.MaxAnalyzeBatch)
	}
	return nil
}

// EnrichAnalysis is the per-result hook used by batch runs (BPM, key, tags).
func EnrichAnalysis(r *core.AnalysisResult) { enrichWithAudioFeatures(r) }

// AnalyzeBatch analyzes paths concurrently, emitting an "analyze-progress"
// event per file, and returns the results finished (all of them unless cancelled).
func (a *App) AnalyzeBatch(batchID string, paths []string) ([]core.AnalysisResult, error) {
	if err := CheckBatchSize(len(paths)); err != nil {
		return nil, err
	}
	ctx, release, err := a.analyzeBatches.Begin(batchID)
	if err != nil {
		return nil, err
	}
	defer release()

	results := core.AnalyzeBatch(ctx, paths, core.DefaultAnalyzeWorkers(), enrichWithAudioFeatures, func(p core.BatchProgress) {
		// Send, not a coalesced send: every row matters, none may be merged.
		a.emitEvent("analyze-progress", map[string]any{
			"batchId": batchID, "done": p.Done, "total": p.Total, "result": p.Result,
		})
	})

	if a.logBuffer != nil {
		lossless, upscaled, errs := 0, 0, 0
		for _, r := range results {
			switch {
			case r.Verdict == "error":
				errs++
			case r.IsTrueLossless:
				lossless++
			default:
				upscaled++
			}
		}
		msg := fmt.Sprintf("Analysis done on %d files: %d lossless, %d upscaled, %d errors", len(results), lossless, upscaled, errs)
		if len(results) < len(paths) {
			msg += fmt.Sprintf(" (cancelled, %d not analyzed)", len(paths)-len(results))
		}
		a.logBuffer.Info(msg)
	}
	return results, nil
}

// CancelAnalyzeBatch stops a running batch; files already analyzed are kept.
func (a *App) CancelAnalyzeBatch(batchID string) bool {
	return a.analyzeBatches.Cancel(batchID)
}

// ExportAnalysisReport saves results as "csv" or "json" through a save dialog.
// It returns the saved path, or an empty string if the user cancelled.
func (a *App) ExportAnalysisReport(format string, results []core.AnalysisResult) (string, error) {
	data, _, ext, err := core.BuildAnalysisReport(results, format)
	if err != nil {
		return "", err
	}
	name := "analysis_report." + ext
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: name,
		Filters:         []runtime.FileFilter{{DisplayName: ext + " files", Pattern: "*." + ext}},
	})
	if err != nil || savePath == "" {
		return "", err
	}
	if err := os.WriteFile(savePath, data, 0o600); err != nil {
		return "", fmt.Errorf("could not write report: %w", err)
	}
	return savePath, nil
}
