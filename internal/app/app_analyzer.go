package app

import (
	"fmt"

	core "github.com/kushiemoon-dev/flacidal-core"
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
