package backend

import (
	"encoding/json"
	"testing"
)

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "One More Time",
			expected: "One More Time",
		},
		{
			name:     "name with invalid chars",
			input:    "Track<>:\"/\\|?*Name",
			expected: "TrackName",
		},
		{
			name:     "name with control chars",
			input:    "Track\x00\x01\x02Name",
			expected: "TrackName",
		},
		{
			name:     "multiple spaces",
			input:    "Track   With    Spaces",
			expected: "Track With Spaces",
		},
		{
			name:     "leading trailing dots spaces",
			input:    "  .Track Name.  ",
			expected: "Track Name",
		},
		{
			name:     "very long name",
			input:    string(make([]byte, 300)),
			expected: "Unknown",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "Unknown",
		},
		{
			name:     "only invalid chars",
			input:    "<>:\"/\\|?*",
			expected: "Unknown",
		},
		{
			name:     "unicode chars",
			input:    "Café au Lait",
			expected: "Café au Lait",
		},
		{
			name:     "artist with feat",
			input:    "Daft Punk feat. Pharrell",
			expected: "Daft Punk feat. Pharrell",
		},
		{
			name:     "file extension like",
			input:    "track.flac",
			expected: "track.flac",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFileName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFileName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseSearchBody(t *testing.T) {
	tests := []struct {
		name          string
		jsonBody      string
		expectedCount int
		expectError   bool
	}{
		{
			name: "data items format",
			jsonBody: `{
				"version": "2.0",
				"data": {
					"items": [
						{"id": 1, "title": "Track 1"},
						{"id": 2, "title": "Track 2"}
					]
				}
			}`,
			expectedCount: 2,
		},
		{
			name: "tracks items format",
			jsonBody: `{
				"tracks": {
					"items": [
						{"id": 3, "title": "Track 3"},
						{"id": 4, "title": "Track 4"},
						{"id": 5, "title": "Track 5"}
					]
				}
			}`,
			expectedCount: 3,
		},
		{
			name: "empty items",
			jsonBody: `{
				"data": {
					"items": []
				}
			}`,
			expectedCount: 0,
		},
		{
			name:        "invalid json",
			jsonBody:    `{invalid}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSearchBody([]byte(tt.jsonBody))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d items, got %d", tt.expectedCount, len(result))
			}
		})
	}
}

func TestFormatCoverUUID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "aa-bb-cc-dd",
			expected: "aa/bb/cc/dd",
		},
		{
			input:    "11-22-33-44-55-66",
			expected: "11/22/33/44/55/66",
		},
		{
			input:    "no-dashes",
			expected: "no/dashes",
		},
		{
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := FormatCoverUUID(tt.input)
			if result != tt.expected {
				t.Errorf("FormatCoverUUID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDownloadResultStruct(t *testing.T) {
	result := DownloadResult{
		TrackID:          12345,
		Title:            "One More Time",
		Artist:           "Daft Punk",
		Album:            "Discovery",
		FilePath:         "/music/One More Time.flac",
		FileSize:         35000000,
		Quality:          "LOSSLESS",
		RequestedQuality: "HI_RES",
		QualityMismatch:  true,
		CoverURL:         "https://example.com/cover.jpg",
		Success:          true,
	}

	if result.TrackID != 12345 {
		t.Errorf("Expected TrackID 12345, got %d", result.TrackID)
	}
	if !result.QualityMismatch {
		t.Error("Expected QualityMismatch to be true")
	}
	if result.FileSize != 35000000 {
		t.Errorf("Expected FileSize 35000000, got %d", result.FileSize)
	}
}

func TestDownloadOptionsStruct(t *testing.T) {
	opts := DownloadOptions{
		Quality:              "HI_RES",
		FileNameFormat:       "{artist} - {title}",
		OrganizeFolders:      true,
		EmbedCover:           true,
		SaveCoverFile:        true,
		AutoAnalyze:          true,
		AutoQualityFallback:  true,
		QualityFallbackOrder: []string{"HI_RES", "LOSSLESS", "HIGH"},
		FirstArtistOnly:      true,
	}

	if opts.Quality != "HI_RES" {
		t.Errorf("Expected Quality 'HI_RES', got %s", opts.Quality)
	}
	if len(opts.QualityFallbackOrder) != 3 {
		t.Errorf("Expected 3 quality fallback options, got %d", len(opts.QualityFallbackOrder))
	}
}

func TestTidalManifestStruct(t *testing.T) {
	jsonStr := `{
		"mimeType": "audio/flac",
		"codecs": "flac",
		"encryptionType": "NONE",
		"urls": ["https://example.com/track.flac"]
	}`

	var manifest TidalManifest
	if err := json.Unmarshal([]byte(jsonStr), &manifest); err != nil {
		t.Fatalf("Failed to parse manifest: %v", err)
	}

	if manifest.MimeType != "audio/flac" {
		t.Errorf("Expected MimeType 'audio/flac', got %s", manifest.MimeType)
	}
	if len(manifest.URLs) != 1 {
		t.Errorf("Expected 1 URL, got %d", len(manifest.URLs))
	}
}

func TestTidalStreamResponseStruct(t *testing.T) {
	jsonStr := `{
		"trackId": 12345,
		"assetId": 67890,
		"audioMode": "STEREO",
		"audioQuality": "LOSSLESS",
		"manifest": "base64encoded",
		"manifestMimeType": "application/vnd.tidal.bts"
	}`

	var resp TidalStreamResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		t.Fatalf("Failed to parse stream response: %v", err)
	}

	if resp.TrackID != 12345 {
		t.Errorf("Expected TrackID 12345, got %d", resp.TrackID)
	}
	if resp.AudioQuality != "LOSSLESS" {
		t.Errorf("Expected AudioQuality 'LOSSLESS', got %s", resp.AudioQuality)
	}
}

func TestStreamInfoStruct(t *testing.T) {
	info := StreamInfo{
		URL:          "https://example.com/track.flac",
		AudioQuality: "HI_RES",
		AudioMode:    "DOLBY_ATMOS",
	}

	if info.AudioQuality != "HI_RES" {
		t.Errorf("Expected AudioQuality 'HI_RES', got %s", info.AudioQuality)
	}
	if info.AudioMode != "DOLBY_ATMOS" {
		t.Errorf("Expected AudioMode 'DOLBY_ATMOS', got %s", info.AudioMode)
	}
}

func TestDownloadedFileInfoStruct(t *testing.T) {
	info := DownloadedFileInfo{
		Path:    "/music/track.flac",
		Name:    "track.flac",
		Size:    35000000,
		ModTime: "2024-01-01T12:00:00Z",
		Title:   "One More Time",
		Artist:  "Daft Punk",
		Album:   "Discovery",
	}

	if info.Name != "track.flac" {
		t.Errorf("Expected Name 'track.flac', got %s", info.Name)
	}
	if info.Size != 35000000 {
		t.Errorf("Expected Size 35000000, got %d", info.Size)
	}
}

func TestQualityFallbackChain(t *testing.T) {
	expected := []string{"HI_RES", "LOSSLESS", "HIGH"}

	if len(qualityFallbackChain) != len(expected) {
		t.Errorf("Expected %d fallback qualities, got %d", len(expected), len(qualityFallbackChain))
		return
	}

	for i, q := range expected {
		if qualityFallbackChain[i] != q {
			t.Errorf("qualityFallbackChain[%d] = %s, want %s", i, qualityFallbackChain[i], q)
		}
	}
}

func BenchmarkSanitizeFileName(b *testing.B) {
	input := "Daft Punk - One More Time (feat. Romanthony) [Remix]"
	for i := 0; i < b.N; i++ {
		SanitizeFileName(input)
	}
}

func BenchmarkParseSearchBody(b *testing.B) {
	jsonBody := []byte(`{
		"data": {
			"items": [
				{"id": 1, "title": "Track 1"},
				{"id": 2, "title": "Track 2"},
				{"id": 3, "title": "Track 3"}
			]
		}
	}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseSearchBody(jsonBody)
	}
}
