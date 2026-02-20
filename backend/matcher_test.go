package backend

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello world"},
		{"Hello & World", "hello and world"},
		{"Test@#$%123", "test123"},
		{"  Multiple   Spaces  ", "multiple spaces"},
		{"UPPERCASE", "uppercase"},
		{"MixEd CaSe", "mixed case"},
		{"", ""},
		{"A", "a"},
		{"Daft Punk", "daft punk"},
		{"AC/DC", "acdc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalize(tt.input)
			if result != tt.expected {
				t.Errorf("normalize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"book", "back", 2},
		{"flac", "flac", 0},
		{"flac", "flacidal", 4},
		{"Daft Punk", "Daft Pank", 1},
		{"Hello World", "Hello", 6},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			result := levenshtein(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestSimilarity(t *testing.T) {
	tests := []struct {
		a, b     string
		minScore float64
		maxScore float64
	}{
		{"", "", 100, 100},
		{"abc", "", 0, 0},
		{"", "abc", 0, 0},
		{"hello", "hello", 100, 100},
		{"hello", "hallo", 60, 100},
		{"Daft Punk", "Daft Punk", 100, 100},
		{"Daft Punk", "Daft Pank", 80, 100},
		{"exact match", "exact match", 100, 100},
		{"completely different", "totally unrelated", 0, 50},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			result := similarity(tt.a, tt.b)
			if result < tt.minScore || result > tt.maxScore {
				t.Errorf("similarity(%q, %q) = %f, want between %f and %f",
					tt.a, tt.b, result, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestCleanTrackTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"One More Time", "One More Time"},
		{"One More Time (feat. Romanthony)", "One More Time"},
		{"Harder Better Faster Stronger (Remix)", "Harder Better Faster Stronger"},
		{"Around the World [Extended]", "Around the World"},
		{"Digital Love (Daft Punk Remix)", "Digital Love"},
		{"Face to Face - Original Mix", "Face to Face - Original Mix"},
		{"", ""},
		{"No Parentheses", "No Parentheses"},
		{"Track (feat. Artist) [Remix]", "Track"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := cleanTrackTitle(tt.input)
			if result != tt.expected {
				t.Errorf("cleanTrackTitle(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildSearchQuery(t *testing.T) {
	tests := []struct {
		title    string
		artist   string
		expected string
	}{
		{"One More Time", "Daft Punk", "track:One More Time artist:Daft Punk"},
		{"Harder Better Faster Stronger", "Daft Punk, Justice", "track:Harder Better Faster Stronger artist:Daft Punk"},
		{"Digital Love", "Daft Punk, Justice, SebastiAn", "track:Digital Love artist:Daft Punk"},
		{"Simple Track", "Artist", "track:Simple Track artist:Artist"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			result := buildSearchQuery(tt.title, tt.artist)
			if result != tt.expected {
				t.Errorf("buildSearchQuery(%q, %q) = %q, want %q",
					tt.title, tt.artist, result, tt.expected)
			}
		})
	}
}

func TestFindBestMatch(t *testing.T) {
	tidalTrack := TidalTrack{
		Title:  "One More Time",
		Artist: "Daft Punk",
		ISRC:   "GBBPF1100001",
	}

	spotifyTracks := []SpotifyTrack{
		{Name: "One More Time", Artists: "Daft Punk", ISRC: "GBBPF1100001"},
		{Name: "One More Time (Remix)", Artists: "Daft Punk", ISRC: ""},
		{Name: "One More Time", Artists: "Unknown Artist", ISRC: ""},
	}

	match, confidence := findBestMatch(tidalTrack, spotifyTracks)

	if match == nil {
		t.Error("Expected a match, got nil")
		return
	}

	if match.Name != "One More Time" {
		t.Errorf("Expected 'One More Time', got %q", match.Name)
	}

	if confidence < 95 {
		t.Errorf("Expected high confidence for exact match, got %d", confidence)
	}
}

func TestFindBestMatch_WithISRCMatch(t *testing.T) {
	tidalTrack := TidalTrack{
		Title:  "Some Track",
		Artist: "Some Artist",
		ISRC:   "USRC17607839",
	}

	spotifyTracks := []SpotifyTrack{
		{Name: "Different Title", Artists: "Different Artist", ISRC: "USRC17607839"},
		{Name: "Some Track", Artists: "Some Artist", ISRC: ""},
	}

	match, confidence := findBestMatch(tidalTrack, spotifyTracks)

	if match == nil {
		t.Error("Expected ISRC match")
		return
	}

	if confidence != 100 {
		t.Errorf("ISRC match should have 100 confidence, got %d", confidence)
	}
}

func TestFindBestMatch_NoMatch(t *testing.T) {
	tidalTrack := TidalTrack{
		Title:  "Completely Unknown Track",
		Artist: "Unknown Artist",
		ISRC:   "",
	}

	spotifyTracks := []SpotifyTrack{
		{Name: "Different Track", Artists: "Different Artist", ISRC: ""},
	}

	_, confidence := findBestMatch(tidalTrack, spotifyTracks)

	if confidence < 70 {
		t.Logf("Low confidence match: %d (expected for different tracks)", confidence)
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		input    []int
		expected int
	}{
		{[]int{1, 2, 3}, 1},
		{[]int{3, 2, 1}, 1},
		{[]int{5}, 5},
		{[]int{-1, 0, 1}, -1},
		{[]int{10, 10, 10}, 10},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := min(tt.input...)
			if result != tt.expected {
				t.Errorf("min(%v) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 2},
		{2, 1, 2},
		{5, 5, 5},
		{-1, 1, 1},
		{0, 0, 0},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := max(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("max(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func BenchmarkNormalize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		normalize("Daft Punk - One More Time (feat. Romanthony)")
	}
}

func BenchmarkLevenshtein(b *testing.B) {
	for i := 0; i < b.N; i++ {
		levenshtein("One More Time", "One More Time (Remix)")
	}
}

func BenchmarkSimilarity(b *testing.B) {
	for i := 0; i < b.N; i++ {
		similarity("Daft Punk", "Daft Pank")
	}
}

func BenchmarkFindBestMatch(b *testing.B) {
	tidalTrack := TidalTrack{
		Title:  "One More Time",
		Artist: "Daft Punk",
	}
	spotifyTracks := []SpotifyTrack{
		{Name: "One More Time", Artists: "Daft Punk"},
		{Name: "One More Time (Remix)", Artists: "Daft Punk"},
		{Name: "Different Track", Artists: "Different Artist"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findBestMatch(tidalTrack, spotifyTracks)
	}
}
