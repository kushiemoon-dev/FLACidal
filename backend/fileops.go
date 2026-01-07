package backend

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RenameResult contains the result of a rename operation
type RenameResult struct {
	OldPath string `json:"oldPath"`
	NewPath string `json:"newPath"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// RenamePreview contains preview information for a rename
type RenamePreview struct {
	OldPath  string `json:"oldPath"`
	OldName  string `json:"oldName"`
	NewName  string `json:"newName"`
	NewPath  string `json:"newPath"`
	HasError bool   `json:"hasError"`
	Error    string `json:"error,omitempty"`
}

// TemplateVars contains variables available for renaming
type TemplateVars struct {
	Title       string
	Artist      string
	Album       string
	TrackNumber string
	Date        string
	Genre       string
	ISRC        string
	FileName    string // Original filename without extension
	Ext         string // File extension
}

// Available templates for renaming
var RenameTemplates = []map[string]string{
	{"name": "Artist - Title", "template": "{artist} - {title}"},
	{"name": "TrackNum - Title", "template": "{tracknumber} - {title}"},
	{"name": "TrackNum - Artist - Title", "template": "{tracknumber} - {artist} - {title}"},
	{"name": "Artist - Album - Title", "template": "{artist} - {album} - {title}"},
	{"name": "Album - TrackNum - Title", "template": "{album} - {tracknumber} - {title}"},
	{"name": "Title", "template": "{title}"},
}

// PreviewRename generates preview of rename operations without actually renaming
func PreviewRename(files []string, template string) []RenamePreview {
	results := make([]RenamePreview, 0, len(files))

	for _, filePath := range files {
		preview := RenamePreview{
			OldPath: filePath,
			OldName: filepath.Base(filePath),
		}

		// Read metadata
		meta, err := ReadFLACMetadata(filePath)
		if err != nil {
			preview.HasError = true
			preview.Error = fmt.Sprintf("Failed to read metadata: %v", err)
			preview.NewName = preview.OldName
			preview.NewPath = filePath
			results = append(results, preview)
			continue
		}

		// Generate new name
		newName, err := applyTemplate(template, meta, filePath)
		if err != nil {
			preview.HasError = true
			preview.Error = fmt.Sprintf("Template error: %v", err)
			preview.NewName = preview.OldName
			preview.NewPath = filePath
			results = append(results, preview)
			continue
		}

		preview.NewName = newName
		preview.NewPath = filepath.Join(filepath.Dir(filePath), newName)

		// Check if new file already exists (and is different file)
		if preview.NewPath != filePath {
			if _, err := os.Stat(preview.NewPath); err == nil {
				preview.HasError = true
				preview.Error = "File already exists"
			}
		}

		results = append(results, preview)
	}

	return results
}

// RenameFiles renames files according to the template
func RenameFiles(files []string, template string) []RenameResult {
	results := make([]RenameResult, 0, len(files))

	for _, filePath := range files {
		result := RenameResult{
			OldPath: filePath,
		}

		// Read metadata
		meta, err := ReadFLACMetadata(filePath)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Failed to read metadata: %v", err)
			result.NewPath = filePath
			results = append(results, result)
			continue
		}

		// Generate new name
		newName, err := applyTemplate(template, meta, filePath)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Template error: %v", err)
			result.NewPath = filePath
			results = append(results, result)
			continue
		}

		newPath := filepath.Join(filepath.Dir(filePath), newName)
		result.NewPath = newPath

		// Skip if same path
		if newPath == filePath {
			result.Success = true
			results = append(results, result)
			continue
		}

		// Check if destination exists
		if _, err := os.Stat(newPath); err == nil {
			result.Success = false
			result.Error = "Destination file already exists"
			results = append(results, result)
			continue
		}

		// Perform rename
		if err := os.Rename(filePath, newPath); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Rename failed: %v", err)
			results = append(results, result)
			continue
		}

		result.Success = true
		results = append(results, result)
	}

	return results
}

// applyTemplate applies a template to generate a new filename
func applyTemplate(template string, meta *FLACMetadata, originalPath string) (string, error) {
	ext := filepath.Ext(originalPath)
	baseName := strings.TrimSuffix(filepath.Base(originalPath), ext)

	// Prepare template variables
	vars := TemplateVars{
		Title:       meta.Title,
		Artist:      meta.Artist,
		Album:       meta.Album,
		TrackNumber: meta.TrackNumber,
		Date:        meta.Date,
		Genre:       meta.Genre,
		ISRC:        meta.ISRC,
		FileName:    baseName,
		Ext:         ext,
	}

	// Default fallbacks
	if vars.Title == "" {
		vars.Title = baseName
	}
	if vars.Artist == "" {
		vars.Artist = "Unknown Artist"
	}
	if vars.Album == "" {
		vars.Album = "Unknown Album"
	}
	if vars.TrackNumber == "" {
		vars.TrackNumber = "00"
	}

	// Pad track number
	if len(vars.TrackNumber) == 1 {
		vars.TrackNumber = "0" + vars.TrackNumber
	}
	// Handle track number with total (e.g., "1/12" -> "01")
	if idx := strings.Index(vars.TrackNumber, "/"); idx > 0 {
		num := vars.TrackNumber[:idx]
		if len(num) == 1 {
			num = "0" + num
		}
		vars.TrackNumber = num
	}

	// Apply template
	result := template
	result = strings.ReplaceAll(result, "{title}", SanitizeFileName(vars.Title))
	result = strings.ReplaceAll(result, "{artist}", SanitizeFileName(vars.Artist))
	result = strings.ReplaceAll(result, "{album}", SanitizeFileName(vars.Album))
	result = strings.ReplaceAll(result, "{tracknumber}", vars.TrackNumber)
	result = strings.ReplaceAll(result, "{date}", SanitizeFileName(vars.Date))
	result = strings.ReplaceAll(result, "{genre}", SanitizeFileName(vars.Genre))
	result = strings.ReplaceAll(result, "{isrc}", vars.ISRC)
	result = strings.ReplaceAll(result, "{filename}", SanitizeFileName(vars.FileName))

	// Ensure we have a valid filename
	result = strings.TrimSpace(result)
	if result == "" {
		result = baseName
	}

	// Add extension
	result += ext

	return result, nil
}

// GetRenameTemplates returns available rename templates
func GetRenameTemplates() []map[string]string {
	return RenameTemplates
}
