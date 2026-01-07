package backend

import (
	"fmt"
	"regexp"
)

// MusicSource defines the interface for music streaming sources
type MusicSource interface {
	// Name returns the source identifier (e.g., "tidal", "qobuz")
	Name() string

	// DisplayName returns human-readable name (e.g., "Tidal", "Qobuz")
	DisplayName() string

	// IsAvailable checks if the source is configured and accessible
	IsAvailable() bool

	// ParseURL extracts content ID and type from a URL
	// Returns: id, contentType ("track", "album", "playlist"), error
	ParseURL(rawURL string) (id string, contentType string, err error)

	// CanHandleURL checks if this source can handle the given URL
	CanHandleURL(rawURL string) bool

	// GetTrack fetches track information by ID
	GetTrack(id string) (*SourceTrack, error)

	// GetAlbum fetches album information with tracks
	GetAlbum(id string) (*SourceAlbum, error)

	// GetPlaylist fetches playlist information with tracks
	GetPlaylist(id string) (*SourcePlaylist, error)

	// GetStreamURL gets the download URL for a track
	GetStreamURL(trackID string, quality string) (string, error)

	// DownloadTrack downloads a track to the specified directory
	DownloadTrack(trackID string, outputDir string, options DownloadOptions) (*DownloadResult, error)
}

// SourceTrack represents a track from any source
type SourceTrack struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Artist      string   `json:"artist"`
	Artists     []string `json:"artists"`
	Album       string   `json:"album"`
	AlbumID     string   `json:"albumId"`
	ISRC        string   `json:"isrc"`
	Duration    int      `json:"duration"` // seconds
	TrackNumber int      `json:"trackNumber"`
	TotalTracks int      `json:"totalTracks"`
	DiscNumber  int      `json:"discNumber"`
	Year        string   `json:"year"`
	Genre       string   `json:"genre"`
	CoverURL    string   `json:"coverUrl"`
	Explicit    bool     `json:"explicit"`
	SourceURL   string   `json:"sourceUrl"`
	Source      string   `json:"source"` // "tidal", "qobuz", etc.
	Quality     string   `json:"quality"`
}

// SourceAlbum represents an album from any source
type SourceAlbum struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Artist      string        `json:"artist"`
	Artists     []string      `json:"artists"`
	Year        string        `json:"year"`
	Genre       string        `json:"genre"`
	CoverURL    string        `json:"coverUrl"`
	TrackCount  int           `json:"trackCount"`
	Tracks      []SourceTrack `json:"tracks"`
	Source      string        `json:"source"`
	SourceURL   string        `json:"sourceUrl"`
	Description string        `json:"description"`
}

// SourcePlaylist represents a playlist from any source
type SourcePlaylist struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Creator     string        `json:"creator"`
	CoverURL    string        `json:"coverUrl"`
	TrackCount  int           `json:"trackCount"`
	Tracks      []SourceTrack `json:"tracks"`
	Source      string        `json:"source"`
	SourceURL   string        `json:"sourceUrl"`
}

// SourceManager manages multiple music sources
type SourceManager struct {
	sources         map[string]MusicSource
	preferredSource string
}

// NewSourceManager creates a new source manager
func NewSourceManager() *SourceManager {
	return &SourceManager{
		sources:         make(map[string]MusicSource),
		preferredSource: "tidal",
	}
}

// RegisterSource adds a source to the manager
func (sm *SourceManager) RegisterSource(source MusicSource) {
	sm.sources[source.Name()] = source
}

// GetSource returns a source by name
func (sm *SourceManager) GetSource(name string) (MusicSource, bool) {
	source, ok := sm.sources[name]
	return source, ok
}

// GetAvailableSources returns all available sources
func (sm *SourceManager) GetAvailableSources() []MusicSource {
	var available []MusicSource
	for _, source := range sm.sources {
		if source.IsAvailable() {
			available = append(available, source)
		}
	}
	return available
}

// DetectSource identifies which source can handle a URL
func (sm *SourceManager) DetectSource(rawURL string) (MusicSource, error) {
	for _, source := range sm.sources {
		if source.CanHandleURL(rawURL) {
			return source, nil
		}
	}
	return nil, fmt.Errorf("no source found for URL: %s", rawURL)
}

// SetPreferredSource sets the default source for searches
func (sm *SourceManager) SetPreferredSource(name string) {
	sm.preferredSource = name
}

// GetPreferredSource returns the preferred source
func (sm *SourceManager) GetPreferredSource() (MusicSource, bool) {
	return sm.GetSource(sm.preferredSource)
}

// SourceInfo contains information about a source for the frontend
type SourceInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Available   bool   `json:"available"`
	URLPattern  string `json:"urlPattern"`
}

// GetSourcesInfo returns info about all registered sources
func (sm *SourceManager) GetSourcesInfo() []SourceInfo {
	var infos []SourceInfo
	for _, source := range sm.sources {
		infos = append(infos, SourceInfo{
			Name:        source.Name(),
			DisplayName: source.DisplayName(),
			Available:   source.IsAvailable(),
		})
	}
	return infos
}

// URL detection helpers
var (
	tidalURLPattern  = regexp.MustCompile(`(?:listen\.)?tidal\.com`)
	qobuzURLPattern  = regexp.MustCompile(`(?:play|open)\.qobuz\.com`)
	deezerURLPattern = regexp.MustCompile(`(?:www\.)?deezer\.com`)
	amazonURLPattern = regexp.MustCompile(`music\.amazon\.`)
)

// DetectSourceFromURL returns the source name based on URL pattern
func DetectSourceFromURL(rawURL string) string {
	switch {
	case tidalURLPattern.MatchString(rawURL):
		return "tidal"
	case qobuzURLPattern.MatchString(rawURL):
		return "qobuz"
	case deezerURLPattern.MatchString(rawURL):
		return "deezer"
	case amazonURLPattern.MatchString(rawURL):
		return "amazon"
	default:
		return ""
	}
}
