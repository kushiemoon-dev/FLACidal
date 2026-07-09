package app

import (
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Characterization tests for the "Matcher Methods" section of app.go.
//
// NOT tested here (documented, not fixed):
//   - The non-nil-matcher branches of MatchPlaylistTracks/MatchSingleTrack
//     call into core.Matcher.MatchTrack, which queries the Spotify API over
//     the network with no injectable seam.

func TestMatchPlaylistTracks_NilMatcher(t *testing.T) {
	a := &App{}
	if got := a.MatchPlaylistTracks([]core.TidalTrack{{ID: 1}}); got != nil {
		t.Errorf("MatchPlaylistTracks() with nil matcher = %v, want nil", got)
	}
}

func TestMatchSingleTrack_NilMatcher(t *testing.T) {
	a := &App{}
	track := core.TidalTrack{ID: 1, Title: "Test"}
	got := a.MatchSingleTrack(track)
	want := core.MatchResult{TidalTrack: track, Matched: false, MatchMethod: "none"}
	if got != want {
		t.Errorf("MatchSingleTrack() with nil matcher = %+v, want %+v", got, want)
	}
}
