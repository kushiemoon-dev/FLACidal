package app

import (
	"errors"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func TestExpandDiscographyRejectsNonSpotifyURL(t *testing.T) {
	if _, err := ExpandDiscography(DiscographyDeps{}, "https://example.com/x"); err == nil {
		t.Fatal("want error")
	}
}

func TestExpandDiscographyNeedsSpotify(t *testing.T) {
	_, err := ExpandDiscography(DiscographyDeps{}, "https://open.spotify.com/artist/abc/discography/all")
	if err == nil || err.Error() != "Spotify client not initialized" {
		t.Fatalf("err = %v", err)
	}
}

func TestQueueDiscographyGuards(t *testing.T) {
	var ce *core.CodedError
	if _, err := QueueDiscography(DiscographyDeps{}, nil, "/out"); !errors.As(err, &ce) || ce.Code != core.CodeQueueNotReady {
		t.Errorf("nil queue: %v", err)
	}
	q := &core.DownloadManager{}
	if _, err := QueueDiscography(DiscographyDeps{Queue: q}, nil, ""); !errors.As(err, &ce) || ce.Code != core.CodeQueueNoOutputDir {
		t.Errorf("empty outputDir: %v", err)
	}
	if _, err := QueueDiscography(DiscographyDeps{Queue: q}, nil, "/out"); err == nil || err.Error() != "Spotify client not initialized" {
		t.Errorf("nil spotify: %v", err)
	}
	if _, err := QueueDiscography(DiscographyDeps{Queue: q, Spotify: &core.SpotifyClient{}}, nil, "/out"); err == nil || err.Error() != "Tidal source not initialized" {
		t.Errorf("nil tidal: %v", err)
	}
}
