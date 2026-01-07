package backend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// ConversionOptions contains options for audio conversion
type ConversionOptions struct {
	Format       string `json:"format"`       // "mp3", "aac", "ogg", "opus", "wav"
	Quality      string `json:"quality"`      // "320k", "256k", "192k", "128k", "V0", "V2"
	OutputDir    string `json:"outputDir"`    // Output directory (empty = same as source)
	DeleteSource bool   `json:"deleteSource"` // Delete source file after conversion
}

// ConversionResult contains the result of a conversion
type ConversionResult struct {
	SourcePath  string `json:"sourcePath"`
	OutputPath  string `json:"outputPath"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
	OutputSize  int64  `json:"outputSize,omitempty"`
	SourceSize  int64  `json:"sourceSize,omitempty"`
}

// ConversionFormat describes an available format
type ConversionFormat struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Extension   string   `json:"extension"`
	Qualities   []string `json:"qualities"`
	Description string   `json:"description"`
}

// Converter handles audio format conversion using FFmpeg
type Converter struct {
	ffmpegPath string
	mu         sync.Mutex
}

// Available conversion formats
var ConversionFormats = []ConversionFormat{
	{
		ID:          "mp3",
		Name:        "MP3",
		Extension:   ".mp3",
		Qualities:   []string{"320k", "256k", "192k", "128k", "V0", "V2"},
		Description: "Most compatible format",
	},
	{
		ID:          "aac",
		Name:        "AAC",
		Extension:   ".m4a",
		Qualities:   []string{"256k", "192k", "128k"},
		Description: "Apple/iTunes format",
	},
	{
		ID:          "ogg",
		Name:        "OGG Vorbis",
		Extension:   ".ogg",
		Qualities:   []string{"q10", "q8", "q6", "q4"},
		Description: "Open source format",
	},
	{
		ID:          "opus",
		Name:        "Opus",
		Extension:   ".opus",
		Qualities:   []string{"256k", "192k", "128k", "96k"},
		Description: "Modern efficient format",
	},
	{
		ID:          "wav",
		Name:        "WAV",
		Extension:   ".wav",
		Qualities:   []string{"pcm"},
		Description: "Uncompressed audio",
	},
}

// NewConverter creates a new Converter instance
func NewConverter() (*Converter, error) {
	// Find FFmpeg in PATH
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		// Try common locations
		commonPaths := []string{
			"/usr/bin/ffmpeg",
			"/usr/local/bin/ffmpeg",
			"/opt/homebrew/bin/ffmpeg",
		}
		for _, p := range commonPaths {
			if _, err := os.Stat(p); err == nil {
				ffmpegPath = p
				break
			}
		}
	}

	if ffmpegPath == "" {
		return nil, fmt.Errorf("FFmpeg not found")
	}

	return &Converter{ffmpegPath: ffmpegPath}, nil
}

// IsAvailable checks if FFmpeg is available
func (c *Converter) IsAvailable() bool {
	return c != nil && c.ffmpegPath != ""
}

// GetFFmpegVersion returns the FFmpeg version string
func (c *Converter) GetFFmpegVersion() (string, error) {
	if !c.IsAvailable() {
		return "", fmt.Errorf("FFmpeg not available")
	}

	cmd := exec.Command(c.ffmpegPath, "-version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Extract first line (version info)
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}
	return "", fmt.Errorf("could not parse version")
}

// GetFormats returns available conversion formats
func (c *Converter) GetFormats() []ConversionFormat {
	return ConversionFormats
}

// Convert converts a single file
func (c *Converter) Convert(sourcePath string, opts ConversionOptions) (*ConversionResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := &ConversionResult{
		SourcePath: sourcePath,
	}

	// Get source file info
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		result.Error = fmt.Sprintf("Source file not found: %v", err)
		return result, nil
	}
	result.SourceSize = sourceInfo.Size()

	// Determine output path
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = filepath.Dir(sourcePath)
	}

	// Get format info
	var format *ConversionFormat
	for _, f := range ConversionFormats {
		if f.ID == opts.Format {
			format = &f
			break
		}
	}
	if format == nil {
		result.Error = fmt.Sprintf("Unknown format: %s", opts.Format)
		return result, nil
	}

	// Build output filename
	baseName := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	outputPath := filepath.Join(outputDir, baseName+format.Extension)
	result.OutputPath = outputPath

	// Check if output already exists
	if _, err := os.Stat(outputPath); err == nil {
		result.Error = "Output file already exists"
		return result, nil
	}

	// Build FFmpeg arguments
	args := []string{
		"-i", sourcePath,
		"-y", // Overwrite output
	}

	// Add format-specific options
	switch opts.Format {
	case "mp3":
		args = append(args, "-codec:a", "libmp3lame")
		if strings.HasPrefix(opts.Quality, "V") {
			// VBR quality
			vbrQ := "0"
			if opts.Quality == "V2" {
				vbrQ = "2"
			}
			args = append(args, "-q:a", vbrQ)
		} else {
			// CBR
			args = append(args, "-b:a", opts.Quality)
		}
	case "aac":
		args = append(args, "-codec:a", "aac", "-b:a", opts.Quality)
	case "ogg":
		args = append(args, "-codec:a", "libvorbis")
		if strings.HasPrefix(opts.Quality, "q") {
			q := strings.TrimPrefix(opts.Quality, "q")
			args = append(args, "-q:a", q)
		} else {
			args = append(args, "-b:a", opts.Quality)
		}
	case "opus":
		args = append(args, "-codec:a", "libopus", "-b:a", opts.Quality)
	case "wav":
		args = append(args, "-codec:a", "pcm_s16le")
	}

	// Add output path
	args = append(args, outputPath)

	// Execute FFmpeg
	cmd := exec.Command(c.ffmpegPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		result.Error = fmt.Sprintf("FFmpeg error: %v - %s", err, string(output))
		return result, nil
	}

	// Get output file info
	outputInfo, err := os.Stat(outputPath)
	if err != nil {
		result.Error = fmt.Sprintf("Output file not created: %v", err)
		return result, nil
	}
	result.OutputSize = outputInfo.Size()
	result.Success = true

	// Delete source if requested
	if opts.DeleteSource && result.Success {
		os.Remove(sourcePath)
	}

	return result, nil
}

// ConvertMultiple converts multiple files
func (c *Converter) ConvertMultiple(files []string, opts ConversionOptions) []ConversionResult {
	results := make([]ConversionResult, 0, len(files))

	for _, file := range files {
		result, _ := c.Convert(file, opts)
		if result != nil {
			results = append(results, *result)
		}
	}

	return results
}

// Global converter instance
var globalConverter *Converter

// GetConverter returns the global converter instance
func GetConverter() *Converter {
	if globalConverter == nil {
		globalConverter, _ = NewConverter()
	}
	return globalConverter
}

// IsConverterAvailable checks if FFmpeg is available
func IsConverterAvailable() bool {
	conv := GetConverter()
	return conv != nil && conv.IsAvailable()
}

// GetFFmpegInfo returns FFmpeg availability and version
func GetFFmpegInfo() map[string]interface{} {
	conv := GetConverter()
	if conv == nil || !conv.IsAvailable() {
		return map[string]interface{}{
			"available": false,
			"path":      "",
			"version":   "",
		}
	}

	version, _ := conv.GetFFmpegVersion()
	return map[string]interface{}{
		"available": true,
		"path":      conv.ffmpegPath,
		"version":   version,
	}
}
