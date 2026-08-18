package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	core "github.com/kushiemoon-dev/flacidal-core"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) IsConverterAvailable() bool {
	return core.IsConverterAvailable()
}

func (a *App) GetFFmpegInfo() map[string]interface{} {
	return core.GetFFmpegInfo()
}

func (a *App) InstallFFmpeg() error {
	progressCh := make(chan core.FFmpegInstallProgress, 10)

	go func() {
		for p := range progressCh {
			runtime.EventsEmit(a.ctx, "ffmpeg-install-progress", p)
		}
	}()

	err := core.InstallFFmpeg(progressCh)
	if err != nil {
		return err
	}

	// So it picks up the newly installed binary's path.
	core.ResetConverter()
	return nil
}

func (a *App) GetFFmpegInstallStatus() map[string]interface{} {
	return map[string]interface{}{
		"systemAvailable": core.IsConverterAvailable(),
		"localInstalled":  core.IsFFmpegInstalledLocally(),
		"localPath":       core.GetLocalFFmpegPath(),
	}
}

func (a *App) GetConversionFormats() []core.ConversionFormat {
	conv := core.GetConverter()
	if conv == nil {
		return []core.ConversionFormat{}
	}
	return conv.GetFormats()
}

func (a *App) ConvertFiles(files []string, format, quality, outputDir string, deleteSource bool) []core.ConversionResult {
	conv := core.GetConverter()
	if conv == nil {
		results := make([]core.ConversionResult, len(files))
		for i, f := range files {
			results[i] = core.ConversionResult{
				SourcePath: f,
				Error:      "FFmpeg not available",
			}
		}
		return results
	}

	opts := core.ConversionOptions{
		Format:       format,
		Quality:      quality,
		OutputDir:    outputDir,
		DeleteSource: deleteSource,
	}

	results := conv.ConvertMultiple(files, opts)

	if a.logBuffer != nil {
		success := 0
		for _, r := range results {
			if r.Success {
				success++
			}
		}
		a.logBuffer.Info(fmt.Sprintf("Converted %d/%d files to %s", success, len(files), format))
	}

	return results
}

func (a *App) SelectFolderForConversion() ([]string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose a folder to convert",
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

func (a *App) ConvertFolder(folderPath, format, quality, outputDir string, deleteSource bool) []core.ConversionResult {
	var files []string
	filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.EqualFold(filepath.Ext(path), ".flac") {
			files = append(files, path)
		}
		return nil
	})

	if len(files) == 0 {
		return nil
	}

	return a.ConvertFiles(files, format, quality, outputDir, deleteSource)
}
