package main

import (
	"fmt"
	"strconv"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// =============================================================================
// Tidal Methods (exposed to frontend)
// =============================================================================

// SetTidalCredentials saves Tidal client credentials
func (a *App) SetTidalCredentials(clientID, clientSecret string) error {
	if a.config == nil {
		a.config = &core.Config{}
	}
	a.config.TidalClientID = clientID
	a.config.TidalClientSecret = clientSecret

	// Initialize client with new credentials
	a.tidalClient = core.NewTidalClient(clientID, clientSecret)

	return core.SaveConfig(a.config)
}

// FetchTidalPlaylist fetches a public playlist from Tidal URL
func (a *App) FetchTidalPlaylist(url string) (*core.TidalPlaylist, error) {
	// Parse URL to get playlist UUID
	id, contentType, err := core.ParseTidalURL(url)
	if err != nil {
		return nil, err
	}

	if contentType != "playlist" {
		return nil, fmt.Errorf("URL is not a playlist (got %s)", contentType)
	}

	return a.downloader.GetPlaylistFromProxy(id)
}

// FetchTidalContent fetches playlist, album, or single track from any Tidal URL
func (a *App) FetchTidalContent(url string) (map[string]interface{}, error) {
	id, contentType, err := core.ParseTidalURL(url)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"type": contentType,
		"id":   id,
	}

	switch contentType {
	case "playlist":
		playlist, err := a.downloader.GetPlaylistFromProxy(id)
		if err != nil {
			return nil, err
		}
		result["title"] = playlist.Title
		result["creator"] = playlist.Creator
		result["coverUrl"] = playlist.CoverURL
		result["tracks"] = playlist.Tracks
		result["trackCount"] = len(playlist.Tracks)

	case "album":
		album, err := a.downloader.GetAlbumFromProxy(id)
		if err != nil {
			return nil, err
		}
		result["title"] = album.Title
		result["creator"] = album.Artist
		result["coverUrl"] = album.CoverURL
		result["tracks"] = album.Tracks
		result["trackCount"] = len(album.Tracks)
		result["albumType"] = album.AlbumType

	case "track":
		if a.downloader == nil {
			return nil, fmt.Errorf("downloader not initialized")
		}
		trackIDInt, convErr := strconv.Atoi(id)
		if convErr != nil {
			return nil, fmt.Errorf("invalid track ID: %s", id)
		}
		track, err := a.downloader.GetTrackAsTidalTrack(trackIDInt)
		if err != nil {
			return nil, err
		}
		result["title"] = track.Title
		result["creator"] = track.Artist
		result["coverUrl"] = track.CoverURL
		result["tracks"] = []core.TidalTrack{*track}
		result["trackCount"] = 1

	case "mix":
		mix, err := a.downloader.GetMixFromProxy(id)
		if err != nil {
			return nil, err
		}
		result["title"] = mix.Title
		result["creator"] = mix.Creator
		result["coverUrl"] = mix.CoverURL
		result["tracks"] = mix.Tracks
		result["trackCount"] = len(mix.Tracks)

	case "artist":
		artist, err := a.tidalClient.GetArtistDiscography(id)
		if err != nil {
			return nil, err
		}
		result["title"] = artist.Name
		result["creator"] = artist.Name
		result["coverUrl"] = artist.PictureURL
		result["albums"] = artist.Albums
		result["albumCount"] = len(artist.Albums)
		result["artistId"] = artist.ID
		result["tracks"] = []core.TidalTrack{} // empty — tracks loaded per album

	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}

	return result, nil
}

// ValidateTidalURL checks if a URL is a valid Tidal URL
func (a *App) ValidateTidalURL(url string) map[string]interface{} {
	id, contentType, err := core.ParseTidalURL(url)
	if err != nil {
		return map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		}
	}
	return map[string]interface{}{
		"valid": true,
		"id":    id,
		"type":  contentType,
	}
}

// RefreshTidalEndpoints forces a re-fetch of the endpoint list from the gist
// and returns the updated list.
func (a *App) RefreshTidalEndpoints() ([]string, error) {
	endpoints, err := core.RefreshTidalEndpoints(true)
	if err != nil {
		return endpoints, err
	}
	// Re-apply endpoints to the downloader unless the user has a full override.
	if len(a.config.TidalHifiEndpoints) == 0 {
		if a.config.TidalCustomEndpoint != "" {
			a.downloader.SetEndpoints(append([]string{a.config.TidalCustomEndpoint}, endpoints...))
		} else {
			a.downloader.SetEndpoints(endpoints)
		}
		a.logBuffer.Info(fmt.Sprintf("Tidal endpoints refreshed: %d endpoints loaded", len(endpoints)))
	}
	return endpoints, nil
}
