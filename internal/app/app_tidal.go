package app

import (
	"fmt"
	"strconv"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func (a *App) SetTidalCredentials(clientID, clientSecret string) error {
	if a.config == nil {
		a.config = &core.Config{}
	}
	a.config.TidalClientID = clientID
	a.config.TidalClientSecret = clientSecret

	a.tidalClient = core.NewTidalClient(clientID, clientSecret)

	return core.SaveConfig(a.config)
}

func (a *App) FetchTidalPlaylist(url string) (*core.TidalPlaylist, error) {
	id, contentType, err := core.ParseTidalURL(url)
	if err != nil {
		return nil, err
	}

	if contentType != "playlist" {
		return nil, fmt.Errorf("URL is not a playlist (got %s)", contentType)
	}

	return a.downloader.GetPlaylistFromProxy(id)
}

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
			return nil, fmt.Errorf("not a valid track ID: %s", id)
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
		result["tracks"] = []core.TidalTrack{} // left empty; tracks get loaded per-album instead

	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}

	return result, nil
}

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

func (a *App) RefreshTidalEndpoints() ([]string, error) {
	endpoints, err := core.RefreshTidalEndpoints(true)
	if err != nil {
		return endpoints, err
	}
	// Push the refreshed endpoints to the downloader, unless the user has set a full override
	if len(a.config.TidalHifiEndpoints) == 0 {
		a.downloader.SetEndpoints(endpoints)
		tidalPriority := core.ResolvePriorityEndpoints(a.config.TidalPriorityEndpoints, a.config.TidalCustomEndpoint)
		applyPriorityEndpoints(a.logBuffer, "Tidal HiFi (downloader)", a.downloader.SetPriorityEndpoints, tidalPriority)
		a.logBuffer.Info(fmt.Sprintf("Tidal endpoints refreshed: %d loaded", len(endpoints)))
	}
	return endpoints, nil
}
