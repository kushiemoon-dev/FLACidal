package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// =============================================================================
// Search Methods (exposed to frontend)
// =============================================================================

// SearchTidal searches for tracks on Tidal
func (a *App) SearchTidal(query string) ([]core.TidalTrack, error) {
	if a.downloader == nil {
		return nil, fmt.Errorf("downloader not initialized")
	}

	results, err := a.downloader.SearchTracks(query, 50)
	if err != nil {
		return nil, err
	}

	// Convert to TidalTrack format for frontend
	tracks := make([]core.TidalTrack, len(results))
	for i, r := range results {
		// Build artist string
		var artists []string
		for _, art := range r.Artists {
			artists = append(artists, art.Name)
		}
		artistStr := ""
		if len(artists) > 0 {
			artistStr = artists[0]
		}
		allArtists := artistStr
		if len(artists) > 1 {
			allArtists = fmt.Sprintf("%s, %s", artists[0], artists[1])
			if len(artists) > 2 {
				allArtists += fmt.Sprintf(" +%d", len(artists)-2)
			}
		}

		// Build cover URL
		coverURL := ""
		if r.Album.Cover != "" {
			coverURL = fmt.Sprintf("https://resources.tidal.com/images/%s/320x320.jpg",
				core.FormatCoverUUID(r.Album.Cover))
		}

		tracks[i] = core.TidalTrack{
			ID:       r.ID,
			Title:    r.Title,
			Artist:   artistStr,
			Artists:  allArtists,
			Album:    r.Album.Title,
			Duration: r.Duration,
			ISRC:     r.ISRC,
			CoverURL: coverURL,
			Explicit: r.Explicit,
		}
	}

	return tracks, nil
}

// SearchTidalAlbums searches for albums on Tidal
func (a *App) SearchTidalAlbums(query string) ([]core.TidalAlbum, error) {
	if a.tidalSource == nil {
		return nil, fmt.Errorf("tidal source not initialized")
	}
	return a.tidalSource.SearchAlbums(query, 20)
}

// SearchTidalArtists searches for artists on Tidal
func (a *App) SearchTidalArtists(query string) ([]core.TidalArtist, error) {
	if a.tidalSource == nil {
		return nil, fmt.Errorf("tidal source not initialized")
	}
	return a.tidalSource.SearchArtists(query, 20)
}

// SearchDeezer searches tracks on the public Deezer API (no auth required).
// Returns up to 30 tracks with ISRC so the orchestrator can find the FLAC.
func (a *App) SearchDeezer(query string) ([]map[string]interface{}, error) {
	if query == "" {
		return []map[string]interface{}{}, nil
	}
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://api.deezer.com/search?q="+url.QueryEscape(query)+"&limit=30", nil)
	if err != nil {
		return nil, fmt.Errorf("deezer search: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; FLACidal/4.6)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("deezer search: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID       int    `json:"id"`
			Title    string `json:"title"`
			Duration int    `json:"duration"`
			ISRC     string `json:"isrc"`
			Preview  string `json:"preview"`
			Artist   struct {
				Name string `json:"name"`
			} `json:"artist"`
			Album struct {
				Title string `json:"title"`
				Cover string `json:"cover_medium"`
			} `json:"album"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("deezer search: %w", err)
	}

	tracks := make([]map[string]interface{}, 0, len(result.Data))
	for _, t := range result.Data {
		tracks = append(tracks, map[string]interface{}{
			"id":       strconv.Itoa(t.ID),
			"title":    t.Title,
			"artist":   t.Artist.Name,
			"album":    t.Album.Title,
			"cover":    t.Album.Cover,
			"duration": t.Duration,
			"isrc":     t.ISRC,
		})
	}
	return tracks, nil
}
