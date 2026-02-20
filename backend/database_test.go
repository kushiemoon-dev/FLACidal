package backend

import (
	"testing"
	"time"
)

func TestDatabaseWithMemory(t *testing.T) {
	t.Run("CachedTrack struct", func(t *testing.T) {
		track := CachedTrack{
			ISRC:           "GBBPF1100001",
			TidalTrackID:   "12345",
			SpotifyTrackID: "spotify-123",
			SpotifyURI:     "spotify:track:123",
			Title:          "One More Time",
			Artist:         "Daft Punk",
			MatchMethod:    "isrc",
			Confidence:     100,
			MatchedAt:      time.Now(),
		}

		if track.ISRC != "GBBPF1100001" {
			t.Errorf("ISRC = %q, want 'GBBPF1100001'", track.ISRC)
		}
		if track.Confidence != 100 {
			t.Errorf("Confidence = %f, want 100", track.Confidence)
		}
	})

	t.Run("DownloadRecord struct", func(t *testing.T) {
		record := DownloadRecord{
			ID:               1,
			TidalContentID:   "playlist-123",
			TidalContentName: "My Playlist",
			ContentType:      "playlist",
			LastDownloadAt:   time.Now(),
			TracksTotal:      10,
			TracksDownloaded: 8,
			TracksFailed:     2,
			CreatedAt:        time.Now(),
		}

		if record.TracksTotal != 10 {
			t.Errorf("TracksTotal = %d, want 10", record.TracksTotal)
		}
		if record.ContentType != "playlist" {
			t.Errorf("ContentType = %q, want 'playlist'", record.ContentType)
		}
	})

	t.Run("MatchFailure struct", func(t *testing.T) {
		failure := MatchFailure{
			ID:            1,
			TidalTrackID:  "track-123",
			ISRC:          "GBBPF1100001",
			Title:         "Unknown Track",
			Artist:        "Unknown Artist",
			Album:         "Unknown Album",
			Reason:        "No match found",
			Attempts:      3,
			LastAttemptAt: time.Now(),
		}

		if failure.Attempts != 3 {
			t.Errorf("Attempts = %d, want 3", failure.Attempts)
		}
	})

	t.Run("HistoryFilter struct", func(t *testing.T) {
		filter := HistoryFilter{
			ContentType: "album",
			Search:      "Discovery",
			Limit:       10,
			Offset:      0,
		}

		if filter.ContentType != "album" {
			t.Errorf("ContentType = %q, want 'album'", filter.ContentType)
		}
		if filter.Limit != 10 {
			t.Errorf("Limit = %d, want 10", filter.Limit)
		}
	})
}

func TestMatchResultStruct(t *testing.T) {
	result := MatchResult{
		TidalTrack: TidalTrack{
			ID:     12345,
			Title:  "One More Time",
			Artist: "Daft Punk",
		},
		SpotifyTrack: &SpotifyTrack{
			ID:      "spotify-123",
			URI:     "spotify:track:123",
			Name:    "One More Time",
			Artists: "Daft Punk",
		},
		Matched:     true,
		MatchMethod: "isrc",
		Confidence:  100,
	}

	if !result.Matched {
		t.Error("Matched should be true")
	}
	if result.MatchMethod != "isrc" {
		t.Errorf("MatchMethod = %q, want 'isrc'", result.MatchMethod)
	}
	if result.Confidence != 100 {
		t.Errorf("Confidence = %d, want 100", result.Confidence)
	}
}

func TestMatchResultNoMatch(t *testing.T) {
	result := MatchResult{
		TidalTrack: TidalTrack{
			ID:     99999,
			Title:  "Unknown Track",
			Artist: "Unknown Artist",
		},
		Matched:     false,
		MatchMethod: "none",
		Confidence:  0,
		Error:       "No matching track found",
	}

	if result.Matched {
		t.Error("Matched should be false")
	}
	if result.MatchMethod != "none" {
		t.Errorf("MatchMethod = %q, want 'none'", result.MatchMethod)
	}
	if result.Error == "" {
		t.Error("Error should not be empty")
	}
}

func TestMatcherStruct(t *testing.T) {
	matcher := Matcher{
		spotify: nil,
		db:      nil,
	}

	if matcher.spotify != nil {
		t.Error("spotify should be nil")
	}
	if matcher.db != nil {
		t.Error("db should be nil")
	}
}

func TestSpotifyTrackStruct(t *testing.T) {
	track := SpotifyTrack{
		ID:      "4uLU6hMCjMI75M1A2tKUQC",
		URI:     "spotify:track:4uLU6hMCjMI75M1A2tKUQC",
		Name:    "One More Time",
		Artists: "Daft Punk",
		Album:   "Discovery",
		ISRC:    "GBBPF1100001",
	}

	if track.ID != "4uLU6hMCjMI75M1A2tKUQC" {
		t.Errorf("ID = %q", track.ID)
	}
	if track.ISRC != "GBBPF1100001" {
		t.Errorf("ISRC = %q", track.ISRC)
	}
}
