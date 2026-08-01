package app

import (
	"fmt"

	core "github.com/kushiemoon-dev/flacidal-core"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// =============================================================================
// Analyzer Methods (exposed to frontend)
// =============================================================================

// AnalyzeFile analyzes a single FLAC file for quality/authenticity
func (a *App) AnalyzeFile(filePath string) (*core.AnalysisResult, error) {
	result, err := core.AnalyzeFLAC(filePath)
	if err != nil {
		return nil, err
	}

	if a.logBuffer != nil {
		a.logBuffer.Info(fmt.Sprintf("Analyzed: %s - %s", result.FileName, result.VerdictLabel))
	}

	return result, nil
}

// SelectFolderForAnalysis opens a directory dialog and returns paths of FLAC
// files within it (recursively), for scanning an existing library for
// upscaled/fake-lossless files after the fact.
func (a *App) SelectFolderForAnalysis() ([]string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select folder to scan for upscaled files",
	})
	if err != nil || dir == "" {
		return nil, err
	}
	files, err := core.ListFLACFiles(dir)
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}
	return paths, nil
}

// AnalyzeMultiple analyzes multiple files
func (a *App) AnalyzeMultiple(filePaths []string) []core.AnalysisResult {
	results := core.AnalyzeMultiple(filePaths)

	if a.logBuffer != nil {
		lossless := 0
		upscaled := 0
		for _, r := range results {
			if r.IsTrueLossless {
				lossless++
			} else if r.Verdict != "error" {
				upscaled++
			}
		}
		a.logBuffer.Info(fmt.Sprintf("Analyzed %d files: %d lossless, %d upscaled", len(results), lossless, upscaled))
	}

	return results
}

// QuickAnalyze performs a fast analysis based on file size heuristics
func (a *App) QuickAnalyze(filePath string) (*core.AnalysisResult, error) {
	return core.QuickAnalyze(filePath)
}
