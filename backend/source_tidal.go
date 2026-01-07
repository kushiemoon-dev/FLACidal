package backend

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// TidalSource implements MusicSource interface for Tidal
type TidalSource struct {
	service     *TidalHifiService
	apiClient   *TidalClient
	available   bool
}

// Tidal URL patterns
var (
	tidalSourcePlaylistRegex = regexp.MustCompile(`tidal\.com/(?:browse/)?playlist/([a-f0-9-]+)`)
	tidalSourceTrackRegex    = regexp.MustCompile(`tidal\.com/(?:browse/)?track/(\d+)`)
	tidalSourceAlbumRegex    = regexp.MustCompile(`tidal\.com/(?:browse/)?album/(\d+)`)
)

// NewTidalSource creates a new Tidal source
func NewTidalSource() *TidalSource {
	service := NewTidalHifiService()
	apiClient := NewTidalClientDefault()

	return &TidalSource{
		service:   service,
		apiClient: apiClient,
		available: true,
	}
}

// Name returns the source identifier
func (t *TidalSource) Name() string {
	return "tidal"
}

// DisplayName returns human-readable name
func (t *TidalSource) DisplayName() string {
	return "Tidal"
}

// IsAvailable checks if the source is enabled
func (t *TidalSource) IsAvailable() bool {
	return t.available
}

// SetAvailable sets the availability status
func (t *TidalSource) SetAvailable(available bool) {
	t.available = available
}

// ParseURL extracts content ID and type from a Tidal URL
func (t *TidalSource) ParseURL(rawURL string) (id string, contentType string, err error) {
	if matches := tidalSourcePlaylistRegex.FindStringSubmatch(rawURL); len(matches) > 1 {
		return matches[1], "playlist", nil
	}
	if matches := tidalSourceTrackRegex.FindStringSubmatch(rawURL); len(matches) > 1 {
		return matches[1], "track", nil
	}
	if matches := tidalSourceAlbumRegex.FindStringSubmatch(rawURL); len(matches) > 1 {
		return matches[1], "album", nil
	}
	return "", "", fmt.Errorf("invalid Tidal URL format")
}

// CanHandleURL checks if this source can handle the given URL
func (t *TidalSource) CanHandleURL(rawURL string) bool {
	_, _, err := t.ParseURL(rawURL)
	return err == nil
}

// GetTrack fetches track information by ID
func (t *TidalSource) GetTrack(id string) (*SourceTrack, error) {
	trackID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid track ID: %s", id)
	}

	track, err := t.service.GetTrackByID(trackID)
	if err != nil {
		return nil, err
	}

	// Build artists list
	artists := make([]string, len(track.Artists))
	for i, a := range track.Artists {
		artists[i] = a.Name
	}
	if len(artists) == 0 && track.Artist.Name != "" {
		artists = []string{track.Artist.Name}
	}

	// Build cover URL
	coverURL := ""
	if track.Album.Cover != "" {
		coverURL = fmt.Sprintf("https://resources.tidal.com/images/%s/640x640.jpg",
			strings.ReplaceAll(track.Album.Cover, "-", "/"))
	}

	return &SourceTrack{
		ID:          id,
		Title:       track.Title,
		Artist:      track.Artist.Name,
		Artists:     artists,
		Album:       track.Album.Title,
		ISRC:        track.ISRC,
		Duration:    track.Duration,
		TrackNumber: track.TrackNumber,
		CoverURL:    coverURL,
		Explicit:    track.Explicit,
		SourceURL:   fmt.Sprintf("https://tidal.com/browse/track/%s", id),
		Source:      "tidal",
		Quality:     t.service.options.Quality,
	}, nil
}

// GetAlbum fetches album information with tracks
func (t *TidalSource) GetAlbum(id string) (*SourceAlbum, error) {
	// Use the API client to get album info
	tidalAlbum, err := t.apiClient.GetAlbum(id)
	if err != nil {
		return nil, err
	}

	// Convert tracks
	tracks := make([]SourceTrack, len(tidalAlbum.Tracks))
	for i, tt := range tidalAlbum.Tracks {
		tracks[i] = SourceTrack{
			ID:          strconv.Itoa(tt.ID),
			Title:       tt.Title,
			Artist:      tt.Artist,
			Album:       tt.Album,
			ISRC:        tt.ISRC,
			Duration:    tt.Duration,
			TrackNumber: tt.TrackNum,
			CoverURL:    tt.CoverURL,
			Explicit:    tt.Explicit,
			SourceURL:   tt.TidalURL,
			Source:      "tidal",
		}
	}

	return &SourceAlbum{
		ID:         id,
		Title:      tidalAlbum.Title,
		Artist:     tidalAlbum.Artist,
		CoverURL:   tidalAlbum.CoverURL,
		TrackCount: len(tracks),
		Tracks:     tracks,
		Source:     "tidal",
		SourceURL:  fmt.Sprintf("https://tidal.com/browse/album/%s", id),
	}, nil
}

// GetPlaylist fetches playlist information with tracks
func (t *TidalSource) GetPlaylist(id string) (*SourcePlaylist, error) {
	tidalPlaylist, err := t.apiClient.GetPlaylist(id)
	if err != nil {
		return nil, err
	}

	// Convert tracks
	tracks := make([]SourceTrack, len(tidalPlaylist.Tracks))
	for i, tt := range tidalPlaylist.Tracks {
		tracks[i] = SourceTrack{
			ID:          strconv.Itoa(tt.ID),
			Title:       tt.Title,
			Artist:      tt.Artist,
			Album:       tt.Album,
			ISRC:        tt.ISRC,
			Duration:    tt.Duration,
			TrackNumber: tt.TrackNum,
			CoverURL:    tt.CoverURL,
			Explicit:    tt.Explicit,
			SourceURL:   tt.TidalURL,
			Source:      "tidal",
		}
	}

	return &SourcePlaylist{
		ID:          tidalPlaylist.UUID,
		Title:       tidalPlaylist.Title,
		Description: tidalPlaylist.Description,
		Creator:     tidalPlaylist.Creator,
		CoverURL:    tidalPlaylist.CoverURL,
		TrackCount:  tidalPlaylist.TrackCount,
		Tracks:      tracks,
		Source:      "tidal",
		SourceURL:   fmt.Sprintf("https://tidal.com/browse/playlist/%s", id),
	}, nil
}

// GetStreamURL gets the download URL for a track
func (t *TidalSource) GetStreamURL(trackID string, quality string) (string, error) {
	id, err := strconv.Atoi(trackID)
	if err != nil {
		return "", fmt.Errorf("invalid track ID: %s", trackID)
	}

	// Temporarily set quality if provided
	if quality != "" {
		oldQuality := t.service.options.Quality
		t.service.options.Quality = quality
		defer func() { t.service.options.Quality = oldQuality }()
	}

	return t.service.GetStreamURL(id)
}

// DownloadTrack downloads a track to the specified directory
func (t *TidalSource) DownloadTrack(trackID string, outputDir string, options DownloadOptions) (*DownloadResult, error) {
	id, err := strconv.Atoi(trackID)
	if err != nil {
		return nil, fmt.Errorf("invalid track ID: %s", trackID)
	}

	// Apply options
	t.service.SetOptions(options)

	return t.service.DownloadTrack(id, outputDir)
}

// GetService returns the underlying TidalHifiService
func (t *TidalSource) GetService() *TidalHifiService {
	return t.service
}

// GetAPIClient returns the underlying TidalClient
func (t *TidalSource) GetAPIClient() *TidalClient {
	return t.apiClient
}
