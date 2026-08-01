package app

import (
	"fmt"

	core "github.com/kushiemoon-dev/flacidal-core"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// =============================================================================
// Analyzer Methods (exposed to frontend)
// =============================================================================

// enrichWithAudioFeatures adds BPM/musical key to an analysis result via
// aubiotrack/keyfinder-cli, embeds them into the file's tags, and (best-effort)
// fills in album/tracknumber/discnumber/year/genre/cover via a Deezer ISRC
// lookup -- for files an untrusted source (Soulseek/Bandcamp/Amazon) delivered
// with incomplete tags before the download-time retag existed, or that only
// got title/artist/isrc from it (see needsRetag in flacidal-core). Every step
// here is best-effort: a missing optional binary, no ISRC to match on, or no
// Deezer match just leaves that part alone rather than failing the analysis.
func enrichWithAudioFeatures(result *core.AnalysisResult) {
	if bpm, err := core.DetectBPM(result.FilePath); err == nil {
		result.BPM = bpm
	}
	if key, err := core.DetectKey(result.FilePath); err == nil {
		result.MusicalKey = key
	}
	if result.BPM > 0 || result.MusicalKey != "" {
		_ = core.NewFLACTagger().EmbedAudioFeatures(result.FilePath, result.BPM, result.MusicalKey)
	}
	core.RetagFromDeezer([]string{result.FilePath})
}

// AnalyzeFile analyzes a single FLAC file for quality/authenticity
func (a *App) AnalyzeFile(filePath string) (*core.AnalysisResult, error) {
	result, err := core.AnalyzeFLAC(filePath)
	if err != nil {
		return nil, err
	}
	enrichWithAudioFeatures(result)

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
	for i := range results {
		if results[i].Verdict != "error" {
			enrichWithAudioFeatures(&results[i])
		}
	}

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
