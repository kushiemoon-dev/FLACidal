package app

import (
	"fmt"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func (a *App) ListDownloadedFiles() ([]core.DownloadedFileInfo, error) {
	folder := a.GetDownloadFolder()
	if folder == "" {
		return []core.DownloadedFileInfo{}, nil
	}

	return core.ListFLACFiles(folder)
}

func (a *App) DeleteFile(path string) error {
	return core.DeleteFile(path)
}

func (a *App) GetFileMetadata(filePath string) (*core.FLACMetadata, error) {
	return core.ReadFLACMetadata(filePath)
}

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

func (a *App) GetRenameTemplates() []map[string]string {
	return core.GetRenameTemplates()
}

func (a *App) PreviewRename(files []string, template string) []core.RenamePreview {
	return core.PreviewRename(files, template)
}

func (a *App) RenameFiles(files []string, template string) []core.RenameResult {
	results := core.RenameFiles(files, template)

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
		a.logBuffer.Info(fmt.Sprintf("Rename finished: %d ok, %d failed", success, failed))
	}

	return results
}
