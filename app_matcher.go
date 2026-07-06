package main

import core "github.com/kushiemoon-dev/flacidal-core"

// =============================================================================
// Matcher Methods (exposed to frontend)
// =============================================================================

// MatchPlaylistTracks matches all tracks from a Tidal playlist to Spotify
func (a *App) MatchPlaylistTracks(tracks []core.TidalTrack) []core.MatchResult {
	if a.matcher == nil {
		return nil
	}
	return a.matcher.MatchPlaylist(tracks)
}

// MatchSingleTrack matches a single track
func (a *App) MatchSingleTrack(track core.TidalTrack) core.MatchResult {
	if a.matcher == nil {
		return core.MatchResult{TidalTrack: track, Matched: false, MatchMethod: "none"}
	}
	return a.matcher.MatchTrack(track)
}
