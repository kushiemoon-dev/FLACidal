package app

import core "github.com/kushiemoon-dev/flacidal-core"

func (a *App) MatchPlaylistTracks(tracks []core.TidalTrack) []core.MatchResult {
	if a.matcher == nil {
		return nil
	}
	return a.matcher.MatchPlaylist(tracks)
}

func (a *App) MatchSingleTrack(track core.TidalTrack) core.MatchResult {
	if a.matcher == nil {
		return core.MatchResult{TidalTrack: track, Matched: false, MatchMethod: "none"}
	}
	return a.matcher.MatchTrack(track)
}
