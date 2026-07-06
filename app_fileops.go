package main

import (
	"fmt"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// =============================================================================
// File Browser Methods (exposed to frontend)
// =============================================================================

// ListDownloadedFiles lists all downloaded FLAC files
func (a *App) ListDownloadedFiles() ([]core.DownloadedFileInfo, error) {
	folder := a.GetDownloadFolder()
	if folder == "" {
		return []core.DownloadedFileInfo{}, nil
	}

	return core.ListFLACFiles(folder)
}

// DeleteFile deletes a file from the filesystem
func (a *App) DeleteFile(path string) error {
	return core.DeleteFile(path)
}

// GetFileMetadata reads and returns metadata from a FLAC file
func (a *App) GetFileMetadata(filePath string) (*core.FLACMetadata, error) {
	return core.ReadFLACMetadata(filePath)
}

// GetFileCoverArt returns cover art as base64 encoded string
func (a *App) GetFileCoverArt(filePath string) (map[string]string, error) {
	base64Data, mimeType, err := core.GetCoverArtBase64(filePath)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"data":     base64Data,
		"mimeType": mimeType,
	}, nil
}

// GetRenameTemplates returns available rename templates
func (a *App) GetRenameTemplates() []map[string]string {
	return core.GetRenameTemplates()
}

// PreviewRename generates a preview of rename operations
func (a *App) PreviewRename(files []string, template string) []core.RenamePreview {
	return core.PreviewRename(files, template)
}

// RenameFiles renames files according to the template
func (a *App) RenameFiles(files []string, template string) []core.RenameResult {
	results := core.RenameFiles(files, template)

	// Log results
	if a.logBuffer != nil {
		success := 0
		failed := 0
		for _, r := range results {
			if r.Success {
				success++
			} else {
				failed++
			}
		}
		a.logBuffer.Info(fmt.Sprintf("Renamed %d files (%d failed)", success, failed))
	}

	return results
}
