package backend

import (
	"testing"
)

func TestDetermineVerdict(t *testing.T) {
	tests := []struct {
		name            string
		spectrumCutoff  int
		expectedCutoff  int
		sampleRate      int
		expectedTrue    bool
		expectedVerdict string
		minConfidence   float64
		maxConfidence   float64
	}{
		{
			name:            "true lossless 44.1kHz",
			spectrumCutoff:  22050,
			expectedCutoff:  22050,
			sampleRate:      44100,
			expectedTrue:    true,
			expectedVerdict: "lossless",
			minConfidence:   80,
			maxConfidence:   100,
		},
		{
			name:            "true lossless 48kHz",
			spectrumCutoff:  24000,
			expectedCutoff:  24000,
			sampleRate:      48000,
			expectedTrue:    true,
			expectedVerdict: "lossless",
			minConfidence:   80,
			maxConfidence:   100,
		},
		{
			name:            "upscaled MP3 128k",
			spectrumCutoff:  16000,
			expectedCutoff:  22050,
			sampleRate:      44100,
			expectedTrue:    false,
			expectedVerdict: "upscaled",
			minConfidence:   90,
			maxConfidence:   100,
		},
		{
			name:            "likely upscaled MP3 192k",
			spectrumCutoff:  17500,
			expectedCutoff:  22050,
			sampleRate:      44100,
			expectedTrue:    false,
			expectedVerdict: "likely_upscaled",
			minConfidence:   80,
			maxConfidence:   95,
		},
		{
			name:            "suspicious high sample rate",
			spectrumCutoff:  19000,
			expectedCutoff:  48000,
			sampleRate:      96000,
			expectedTrue:    false,
			expectedVerdict: "likely_upscaled",
			minConfidence:   60,
			maxConfidence:   80,
		},
		{
			name:            "good ratio near 1.0",
			spectrumCutoff:  20000,
			expectedCutoff:  22050,
			sampleRate:      44100,
			expectedTrue:    true,
			expectedVerdict: "lossless",
			minConfidence:   80,
			maxConfidence:   100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AnalysisResult{
				SpectrumCutoff: tt.spectrumCutoff,
				ExpectedCutoff: tt.expectedCutoff,
				SampleRate:     tt.sampleRate,
			}

			determineVerdict(result)

			if result.IsTrueLossless != tt.expectedTrue {
				t.Errorf("IsTrueLossless = %v, want %v", result.IsTrueLossless, tt.expectedTrue)
			}

			if result.Verdict != tt.expectedVerdict {
				t.Errorf("Verdict = %q, want %q", result.Verdict, tt.expectedVerdict)
			}

			if result.Confidence < tt.minConfidence || result.Confidence > tt.maxConfidence {
				t.Errorf("Confidence = %f, want between %f and %f",
					result.Confidence, tt.minConfidence, tt.maxConfidence)
			}
		})
	}
}

func TestParseFFmpegOutput(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		expectedMin int
		expectedMax int
	}{
		{
			name: "low RMS - likely MP3",
			output: `Stream #0:0: Audio: flac
			RMS level dB: -45.5`,
			expectedMin: 15000,
			expectedMax: 17000,
		},
		{
			name: "medium RMS - possible lossy",
			output: `Stream #0:0: Audio: flac
			RMS level dB: -35.2`,
			expectedMin: 17000,
			expectedMax: 20000,
		},
		{
			name: "high RMS - likely lossless",
			output: `Stream #0:0: Audio: flac
			RMS level dB: -20.1`,
			expectedMin: 20000,
			expectedMax: 25000,
		},
		{
			name:        "no RMS info",
			output:      `Stream #0:0: Audio: flac`,
			expectedMin: 15000,
			expectedMax: 23000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFFmpegOutput(tt.output)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result < tt.expectedMin || result > tt.expectedMax {
				t.Errorf("parseFFmpegOutput() = %d, want between %d and %d",
					result, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestAnalysisResultStruct(t *testing.T) {
	result := AnalysisResult{
		FilePath:       "/music/track.flac",
		FileName:       "track.flac",
		IsTrueLossless: true,
		Confidence:     95.5,
		SpectrumCutoff: 22050,
		ExpectedCutoff: 22050,
		Verdict:        "lossless",
		VerdictLabel:   "True Lossless",
		Details:        "Full frequency spectrum up to 22050 Hz",
		SampleRate:     44100,
		BitDepth:       16,
	}

	if result.FilePath != "/music/track.flac" {
		t.Errorf("FilePath = %q, want '/music/track.flac'", result.FilePath)
	}
	if !result.IsTrueLossless {
		t.Error("IsTrueLossless should be true")
	}
	if result.Confidence != 95.5 {
		t.Errorf("Confidence = %f, want 95.5", result.Confidence)
	}
	if result.SampleRate != 44100 {
		t.Errorf("SampleRate = %d, want 44100", result.SampleRate)
	}
}

func TestAnalyzeWithoutFFmpeg(t *testing.T) {
	tests := []struct {
		name            string
		bitDepth        int
		sampleRate      int
		expectedTrue    bool
		expectedVerdict string
	}{
		{
			name:            "24-bit should be lossless",
			bitDepth:        24,
			sampleRate:      96000,
			expectedTrue:    true,
			expectedVerdict: "lossless",
		},
		{
			name:            "16-bit unknown",
			bitDepth:        16,
			sampleRate:      44100,
			expectedTrue:    true,
			expectedVerdict: "unknown",
		},
		{
			name:            "32-bit hi-res",
			bitDepth:        32,
			sampleRate:      192000,
			expectedTrue:    true,
			expectedVerdict: "lossless",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AnalysisResult{}
			meta := &FLACMetadata{
				BitDepth:   tt.bitDepth,
				SampleRate: tt.sampleRate,
			}

			_, err := analyzeWithoutFFmpeg(result, meta)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.IsTrueLossless != tt.expectedTrue {
				t.Errorf("IsTrueLossless = %v, want %v", result.IsTrueLossless, tt.expectedTrue)
			}

			if result.Verdict != tt.expectedVerdict {
				t.Errorf("Verdict = %q, want %q", result.Verdict, tt.expectedVerdict)
			}
		})
	}
}

func TestMinInt(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{-5, 0, -5},
		{100, 100, 100},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := minInt(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("minInt(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestRound(t *testing.T) {
	tests := []struct {
		input    float64
		expected int
	}{
		{1.4, 1},
		{1.5, 2},
		{1.6, 2},
		{-1.5, -2},
		{0.0, 0},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := round(tt.input)
			if result != tt.expected {
				t.Errorf("round(%f) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func BenchmarkDetermineVerdict(b *testing.B) {
	result := &AnalysisResult{
		SpectrumCutoff: 22050,
		ExpectedCutoff: 22050,
		SampleRate:     44100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		determineVerdict(result)
	}
}
