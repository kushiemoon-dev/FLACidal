package backend

import (
	"testing"
)

func TestParseTidalURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		expectedID   string
		expectedType string
		expectError  bool
	}{
		{
			name:         "playlist URL",
			url:          "https://tidal.com/browse/playlist/a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			expectedID:   "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			expectedType: "playlist",
		},
		{
			name:         "playlist URL without browse",
			url:          "https://tidal.com/playlist/a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			expectedID:   "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			expectedType: "playlist",
		},
		{
			name:         "track URL",
			url:          "https://tidal.com/browse/track/12345678",
			expectedID:   "12345678",
			expectedType: "track",
		},
		{
			name:         "track URL without browse",
			url:          "https://tidal.com/track/12345678",
			expectedID:   "12345678",
			expectedType: "track",
		},
		{
			name:         "album URL",
			url:          "https://tidal.com/browse/album/87654321",
			expectedID:   "87654321",
			expectedType: "album",
		},
		{
			name:         "album URL without browse",
			url:          "https://tidal.com/album/87654321",
			expectedID:   "87654321",
			expectedType: "album",
		},
		{
			name:         "artist URL",
			url:          "https://tidal.com/browse/artist/11112222",
			expectedID:   "11112222",
			expectedType: "artist",
		},
		{
			name:         "mix URL",
			url:          "https://tidal.com/browse/mix/abc123XYZ",
			expectedID:   "abc123XYZ",
			expectedType: "mix",
		},
		{
			name:        "invalid URL",
			url:         "https://spotify.com/track/123",
			expectError: true,
		},
		{
			name:        "malformed URL",
			url:         "not a url at all",
			expectError: true,
		},
		{
			name:        "tidal URL without type",
			url:         "https://tidal.com/",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, contentType, err := ParseTidalURL(tt.url)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for URL %q, got none", tt.url)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for URL %q: %v", tt.url, err)
				return
			}

			if id != tt.expectedID {
				t.Errorf("ParseTidalURL(%q) id = %q, want %q", tt.url, id, tt.expectedID)
			}

			if contentType != tt.expectedType {
				t.Errorf("ParseTidalURL(%q) type = %q, want %q", tt.url, contentType, tt.expectedType)
			}
		})
	}
}

func TestIsTidalPlaylistURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://tidal.com/browse/playlist/abc-123", true},
		{"https://tidal.com/playlist/abc-123", true},
		{"https://tidal.com/browse/track/123", false},
		{"https://tidal.com/browse/album/456", false},
		{"https://spotify.com/playlist/abc", false},
		{"not a url", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := IsTidalPlaylistURL(tt.url)
			if result != tt.expected {
				t.Errorf("IsTidalPlaylistURL(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestFormatTidalImageURL(t *testing.T) {
	tests := []struct {
		imageID  string
		expected string
	}{
		{
			imageID:  "abc-def-ghi-123",
			expected: "https://resources.tidal.com/images/abc/def/ghi/123/640x640.jpg",
		},
		{
			imageID:  "11-22-33",
			expected: "https://resources.tidal.com/images/11/22/33/640x640.jpg",
		},
		{
			imageID:  "",
			expected: "",
		},
		{
			imageID:  "simpleid",
			expected: "https://resources.tidal.com/images/simpleid/640x640.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.imageID, func(t *testing.T) {
			result := formatTidalImageURL(tt.imageID)
			if result != tt.expected {
				t.Errorf("formatTidalImageURL(%q) = %q, want %q", tt.imageID, result, tt.expected)
			}
		})
	}
}

func TestArtistImageURLs(t *testing.T) {
	tests := []struct {
		name       string
		pictureID  string
		expectNil  bool
		expectKeys []string
	}{
		{
			name:       "valid picture ID",
			pictureID:  "aa-bb-cc-dd",
			expectNil:  false,
			expectKeys: []string{"profile", "profile_hires", "banner"},
		},
		{
			name:      "empty picture ID",
			pictureID: "",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ArtistImageURLs(tt.pictureID)

			if tt.expectNil {
				if result != nil {
					t.Errorf("Expected nil for empty picture ID, got %v", result)
				}
				return
			}

			if result == nil {
				t.Error("Expected non-nil result")
				return
			}

			for _, key := range tt.expectKeys {
				if _, ok := result[key]; !ok {
					t.Errorf("Missing expected key %q in result", key)
				}
			}

			if result["profile"] != "https://resources.tidal.com/images/aa/bb/cc/dd/640x640.jpg" {
				t.Errorf("Unexpected profile URL: %s", result["profile"])
			}
		})
	}
}

func TestTidalTrackStruct(t *testing.T) {
	track := TidalTrack{
		ID:         12345,
		Title:      "One More Time",
		Artist:     "Daft Punk",
		Artists:    "Daft Punk",
		Album:      "Discovery",
		AlbumID:    67890,
		ISRC:       "GBBPF1100001",
		Duration:   320,
		TrackNum:   1,
		CoverURL:   "https://example.com/cover.jpg",
		Explicit:   false,
		TidalURL:   "https://tidal.com/browse/track/12345",
		Available:  true,
		Copyright:  "2024 Daft Life",
		Label:      "Daft Life",
		Popularity: 85,
	}

	if track.ID != 12345 {
		t.Errorf("Expected ID 12345, got %d", track.ID)
	}
	if track.Title != "One More Time" {
		t.Errorf("Expected title 'One More Time', got %s", track.Title)
	}
	if track.Duration != 320 {
		t.Errorf("Expected duration 320, got %d", track.Duration)
	}
}

func TestTidalPlaylistStruct(t *testing.T) {
	playlist := TidalPlaylist{
		UUID:        "abc-123-def",
		Title:       "Discovery Album",
		Description: "Daft Punk's Discovery",
		Creator:     "User123",
		CoverURL:    "https://example.com/cover.jpg",
		TrackCount:  14,
		Tracks: []TidalTrack{
			{ID: 1, Title: "Track 1"},
			{ID: 2, Title: "Track 2"},
		},
	}

	if playlist.UUID != "abc-123-def" {
		t.Errorf("Expected UUID 'abc-123-def', got %s", playlist.UUID)
	}
	if len(playlist.Tracks) != 2 {
		t.Errorf("Expected 2 tracks, got %d", len(playlist.Tracks))
	}
}

func TestTidalAlbumStruct(t *testing.T) {
	album := TidalAlbum{
		ID:          12345,
		Title:       "Discovery",
		Artist:      "Daft Punk",
		ReleaseDate: "2001-02-26",
		TrackCount:  14,
		CoverURL:    "https://example.com/cover.jpg",
		AlbumType:   "ALBUM",
		Copyright:   "2001 Daft Life",
		Label:       "Daft Life",
		Tracks:      []TidalTrack{{ID: 1}, {ID: 2}},
	}

	if album.ID != 12345 {
		t.Errorf("Expected ID 12345, got %d", album.ID)
	}
	if album.AlbumType != "ALBUM" {
		t.Errorf("Expected AlbumType 'ALBUM', got %s", album.AlbumType)
	}
}

func TestTidalArtistStruct(t *testing.T) {
	artist := TidalArtist{
		ID:         12345,
		Name:       "Daft Punk",
		PictureURL: "https://example.com/picture.jpg",
		Albums: []TidalAlbum{
			{ID: 1, Title: "Discovery"},
			{ID: 2, Title: "Random Access Memories"},
		},
	}

	if artist.ID != 12345 {
		t.Errorf("Expected ID 12345, got %d", artist.ID)
	}
	if len(artist.Albums) != 2 {
		t.Errorf("Expected 2 albums, got %d", len(artist.Albums))
	}
}

func BenchmarkParseTidalURL(b *testing.B) {
	url := "https://tidal.com/browse/playlist/a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	for i := 0; i < b.N; i++ {
		ParseTidalURL(url)
	}
}

func BenchmarkFormatTidalImageURL(b *testing.B) {
	imageID := "aa-bb-cc-dd-ee-ff-11-22"
	for i := 0; i < b.N; i++ {
		formatTidalImageURL(imageID)
	}
}
