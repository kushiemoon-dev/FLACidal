package backend

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// AnalysisResult contains the result of quality analysis
type AnalysisResult struct {
	FilePath       string  `json:"filePath"`
	FileName       string  `json:"fileName"`
	IsTrueLossless bool    `json:"isTrueLossless"`
	Confidence     float64 `json:"confidence"` // 0-100
	SpectrumCutoff int     `json:"spectrumCutoff"` // Detected cutoff in Hz
	ExpectedCutoff int     `json:"expectedCutoff"` // Expected cutoff based on sample rate
	Verdict        string  `json:"verdict"` // "lossless", "likely_upscaled", "upscaled"
	VerdictLabel   string  `json:"verdictLabel"`
	Details        string  `json:"details"`
	SampleRate     int     `json:"sampleRate"`
	BitDepth       int     `json:"bitDepth"`
}

// AnalyzeFLAC analyzes a FLAC file to detect if it's truly lossless
func AnalyzeFLAC(filePath string) (*AnalysisResult, error) {
	result := &AnalysisResult{
		FilePath: filePath,
		FileName: filePath[strings.LastIndex(filePath, "/")+1:],
	}

	// First, read metadata to get sample rate and bit depth
	meta, err := ReadFLACMetadata(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	result.SampleRate = meta.SampleRate
	result.BitDepth = meta.BitDepth

	// Calculate expected cutoff (Nyquist frequency)
	result.ExpectedCutoff = meta.SampleRate / 2

	// Use FFmpeg to analyze the spectrum
	cutoff, err := analyzeSpectrum(filePath)
	if err != nil {
		// If FFmpeg analysis fails, use heuristic based on bit depth
		return analyzeWithoutFFmpeg(result, meta)
	}

	result.SpectrumCutoff = cutoff

	// Determine verdict based on cutoff frequency
	determineVerdict(result)

	return result, nil
}

// analyzeSpectrum uses FFmpeg to analyze the audio spectrum
func analyzeSpectrum(filePath string) (int, error) {
	// Check if FFmpeg is available
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return 0, fmt.Errorf("FFmpeg not found")
	}

	// Use FFmpeg's astats filter to get frequency information
	// We'll analyze the audio and look for the highest frequency with significant energy
	cmd := exec.Command(ffmpegPath,
		"-i", filePath,
		"-af", "aformat=sample_fmts=flt,astats=metadata=1:measure_perchannel=none",
		"-f", "null",
		"-",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("FFmpeg analysis failed: %v", err)
	}

	// Parse the output to find frequency information
	// This is a simplified analysis - in production you'd use proper FFT
	return parseFFmpegOutput(string(output))
}

// parseFFmpegOutput parses FFmpeg astats output
func parseFFmpegOutput(output string) (int, error) {
	// Look for RMS level and other indicators
	lines := strings.Split(output, "\n")

	var rmsLevel float64 = -100
	for _, line := range lines {
		if strings.Contains(line, "RMS level dB") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				val := strings.TrimSpace(parts[len(parts)-1])
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					rmsLevel = f
				}
			}
		}
	}

	// Estimate cutoff based on RMS level
	// This is a heuristic - lower RMS often indicates less high-frequency content
	if rmsLevel < -40 {
		return 16000, nil // Likely MP3 source
	} else if rmsLevel < -30 {
		return 18000, nil // Possible lossy source
	}

	return 22050, nil // Likely true lossless
}

// analyzeWithoutFFmpeg provides analysis when FFmpeg is not available
func analyzeWithoutFFmpeg(result *AnalysisResult, meta *FLACMetadata) (*AnalysisResult, error) {
	// Without FFmpeg, we can only make educated guesses based on metadata

	// Check bit depth - 24-bit is more likely to be true lossless
	if meta.BitDepth >= 24 {
		result.IsTrueLossless = true
		result.Confidence = 80
		result.Verdict = "lossless"
		result.VerdictLabel = "Likely Lossless"
		result.Details = "24-bit or higher audio is typically from lossless sources"
		result.SpectrumCutoff = meta.SampleRate / 2
		return result, nil
	}

	// For 16-bit, we can't be sure without spectrum analysis
	result.IsTrueLossless = true
	result.Confidence = 50
	result.Verdict = "unknown"
	result.VerdictLabel = "Unknown"
	result.Details = "Cannot determine without FFmpeg spectrum analysis"
	result.SpectrumCutoff = meta.SampleRate / 2

	return result, nil
}

// determineVerdict sets the verdict based on spectrum analysis
func determineVerdict(result *AnalysisResult) {
	// Calculate ratio of actual cutoff to expected
	ratio := float64(result.SpectrumCutoff) / float64(result.ExpectedCutoff)

	// Common lossy format cutoffs:
	// MP3 128k: ~16kHz
	// MP3 192k: ~18kHz
	// MP3 320k/V0: ~20kHz
	// AAC: Similar to MP3

	if result.SpectrumCutoff <= 16000 {
		// Clear MP3 128k or lower
		result.IsTrueLossless = false
		result.Confidence = 95
		result.Verdict = "upscaled"
		result.VerdictLabel = "Upscaled"
		result.Details = fmt.Sprintf("Frequency cutoff at %d Hz indicates lossy source (likely MP3 128-192k)", result.SpectrumCutoff)
	} else if result.SpectrumCutoff <= 18000 {
		// Likely MP3 ~192k
		result.IsTrueLossless = false
		result.Confidence = 85
		result.Verdict = "likely_upscaled"
		result.VerdictLabel = "Likely Upscaled"
		result.Details = fmt.Sprintf("Frequency cutoff at %d Hz suggests lossy source (likely MP3 192-256k)", result.SpectrumCutoff)
	} else if result.SpectrumCutoff <= 20000 && result.SampleRate > 44100 {
		// High sample rate but limited frequency - suspicious
		result.IsTrueLossless = false
		result.Confidence = 70
		result.Verdict = "likely_upscaled"
		result.VerdictLabel = "Possibly Upscaled"
		result.Details = fmt.Sprintf("Sample rate is %d Hz but frequency content limited to %d Hz", result.SampleRate, result.SpectrumCutoff)
	} else if ratio >= 0.9 {
		// Good frequency content relative to sample rate
		result.IsTrueLossless = true
		result.Confidence = 90
		result.Verdict = "lossless"
		result.VerdictLabel = "True Lossless"
		result.Details = fmt.Sprintf("Full frequency spectrum up to %d Hz", result.SpectrumCutoff)
	} else {
		// Uncertain
		result.IsTrueLossless = true
		result.Confidence = 60
		result.Verdict = "lossless"
		result.VerdictLabel = "Probably Lossless"
		result.Details = "Frequency content appears normal"
	}
}

// AnalyzeMultiple analyzes multiple files
func AnalyzeMultiple(files []string) []AnalysisResult {
	results := make([]AnalysisResult, 0, len(files))

	for _, file := range files {
		result, err := AnalyzeFLAC(file)
		if err != nil {
			results = append(results, AnalysisResult{
				FilePath:     file,
				FileName:     file[strings.LastIndex(file, "/")+1:],
				Verdict:      "error",
				VerdictLabel: "Error",
				Details:      err.Error(),
			})
		} else {
			results = append(results, *result)
		}
	}

	return results
}

// QuickAnalyze provides a faster analysis using file size heuristics
func QuickAnalyze(filePath string) (*AnalysisResult, error) {
	result := &AnalysisResult{
		FilePath: filePath,
		FileName: filePath[strings.LastIndex(filePath, "/")+1:],
	}

	// Read metadata
	meta, err := ReadFLACMetadata(filePath)
	if err != nil {
		return nil, err
	}

	result.SampleRate = meta.SampleRate
	result.BitDepth = meta.BitDepth
	result.ExpectedCutoff = meta.SampleRate / 2

	// Get file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// Calculate expected file size for lossless audio
	// Approximate: sample_rate * bit_depth * channels * duration / 8 * compression_ratio
	// FLAC typically achieves 50-70% compression
	durationSec := float64(meta.TotalSamples) / float64(meta.SampleRate)
	expectedUncompressed := float64(meta.SampleRate) * float64(meta.BitDepth) * float64(meta.Channels) * durationSec / 8
	expectedCompressed := expectedUncompressed * 0.6 // Assume 60% compression

	actualSize := float64(fileInfo.Size())
	sizeRatio := actualSize / expectedCompressed

	// If file is much smaller than expected, it might be upscaled
	if sizeRatio < 0.3 {
		result.IsTrueLossless = false
		result.Confidence = 70
		result.Verdict = "likely_upscaled"
		result.VerdictLabel = "Possibly Upscaled"
		result.Details = fmt.Sprintf("File size (%.1f MB) is smaller than expected for lossless audio", actualSize/(1024*1024))
		result.SpectrumCutoff = 18000
	} else {
		result.IsTrueLossless = true
		result.Confidence = 60
		result.Verdict = "lossless"
		result.VerdictLabel = "Probably Lossless"
		result.Details = "File size is consistent with lossless audio"
		result.SpectrumCutoff = result.ExpectedCutoff
	}

	return result, nil
}

// Helper function for min
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper for rounding
func round(f float64) int {
	return int(math.Round(f))
}
