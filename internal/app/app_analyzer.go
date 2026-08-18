package app

import (
	"fmt"

	core "github.com/kushiemoon-dev/flacidal-core"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// enrichWithAudioFeatures fills in BPM and musical key on an analysis result
// using aubiotrack/keyfinder-cli, writes them into the file's tags, and, on a
// best-effort basis, backfills album/tracknumber/discnumber/year/genre/cover
// via a Deezer ISRC lookup. This covers files that an untrusted source
// (Soulseek/Bandcamp/Amazon) delivered with incomplete tags before the
// download-time retag existed, or that only picked up title/artist/isrc from
// it (see needsRetag in flacidal-core). Nothing here is required to succeed:
// a missing optional binary, an absent ISRC, or no Deezer match simply skips
// that piece instead of failing the whole analysis.
func enrichWithAudioFeatures(result *core.AnalysisResult) {
	if bpm, err := core.DetectBPM(result.FilePath); err == nil {
		result.BPM = bpm
	}
	if key, err := core.DetectKey(result.FilePath); err == nil {
		result.MusicalKey = core.ToCamelotKey(key)
	}
	if result.BPM > 0 || result.MusicalKey != "" {
		_ = core.NewFLACTagger().EmbedAudioFeatures(result.FilePath, result.BPM, result.MusicalKey)
	}
	core.RetagFromDeezer([]string{result.FilePath})
}

func (a *App) AnalyzeFile(filePath string) (*core.AnalysisResult, error) {
	result, err := core.AnalyzeFLAC(filePath)
	if err != nil {
		return nil, err
	}
	enrichWithAudioFeatures(result)

	if a.logBuffer != nil {
		a.logBuffer.Info(fmt.Sprintf("Analysis done: %s - %s", result.FileName, result.VerdictLabel))
	}

	return result, nil
}

// Useful for scanning an existing library for upscaled or fake-lossless files after the fact.
func (a *App) SelectFolderForAnalysis() ([]string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose a folder to scan for upscaled files",
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
		a.logBuffer.Info(fmt.Sprintf("Analysis done on %d files: %d lossless, %d upscaled", len(results), lossless, upscaled))
	}

	return results
}

// A size heuristic, not full decoding — faster but less rigorous than AnalyzeFile.
func (a *App) QuickAnalyze(filePath string) (*core.AnalysisResult, error) {
	return core.QuickAnalyze(filePath)
}
