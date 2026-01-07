package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// LyricsClient handles fetching lyrics from LRCLIB
type LyricsClient struct {
	httpClient *http.Client
	baseURL    string
}

// Lyrics contains lyrics data
type Lyrics struct {
	Plain     string `json:"plain"`     // Plain text lyrics
	Synced    string `json:"synced"`    // LRC format synced lyrics
	Source    string `json:"source"`    // Source (e.g., "lrclib")
	HasSynced bool   `json:"hasSynced"` // Whether synced lyrics are available
	TrackName string `json:"trackName"` // Track name from API
	ArtistName string `json:"artistName"` // Artist name from API
	AlbumName  string `json:"albumName"`  // Album name from API
	Duration   int    `json:"duration"`   // Duration in seconds
}

// lrclibResponse represents the API response from LRCLIB
type lrclibResponse struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	TrackName    string `json:"trackName"`
	ArtistName   string `json:"artistName"`
	AlbumName    string `json:"albumName"`
	Duration     int    `json:"duration"`
	Instrumental bool   `json:"instrumental"`
	PlainLyrics  string `json:"plainLyrics"`
	SyncedLyrics string `json:"syncedLyrics"`
}

// NewLyricsClient creates a new lyrics client
func NewLyricsClient() *LyricsClient {
	return &LyricsClient{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		baseURL: "https://lrclib.net/api",
	}
}

// SearchLyrics searches for lyrics by title, artist and optionally duration
func (lc *LyricsClient) SearchLyrics(title, artist string, durationSec int) (*Lyrics, error) {
	// First try exact match with GET endpoint
	lyrics, err := lc.getExactMatch(title, artist, durationSec)
	if err == nil && lyrics != nil {
		return lyrics, nil
	}

	// Fall back to search endpoint
	return lc.searchFallback(title, artist)
}

// getExactMatch tries to get an exact match using the GET endpoint
func (lc *LyricsClient) getExactMatch(title, artist string, durationSec int) (*Lyrics, error) {
	params := url.Values{}
	params.Set("track_name", title)
	params.Set("artist_name", artist)
	if durationSec > 0 {
		params.Set("duration", fmt.Sprintf("%d", durationSec))
	}

	reqURL := fmt.Sprintf("%s/get?%s", lc.baseURL, params.Encode())

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "FLACidal/1.0 (https://github.com/flacidal)")

	resp, err := lc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil // Not found, try fallback
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("LRCLIB API error: %d", resp.StatusCode)
	}

	var result lrclibResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return lc.convertResponse(&result), nil
}

// searchFallback uses the search endpoint for fuzzy matching
func (lc *LyricsClient) searchFallback(title, artist string) (*Lyrics, error) {
	query := fmt.Sprintf("%s %s", artist, title)
	params := url.Values{}
	params.Set("q", query)

	reqURL := fmt.Sprintf("%s/search?%s", lc.baseURL, params.Encode())

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "FLACidal/1.0 (https://github.com/flacidal)")

	resp, err := lc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("LRCLIB search error: %d", resp.StatusCode)
	}

	var results []lrclibResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no lyrics found for %s - %s", artist, title)
	}

	// Find best match - prefer results with synced lyrics
	var bestMatch *lrclibResponse
	for i := range results {
		r := &results[i]
		// Skip instrumental tracks
		if r.Instrumental {
			continue
		}
		// Check if this is a better match
		if bestMatch == nil {
			bestMatch = r
		} else if r.SyncedLyrics != "" && bestMatch.SyncedLyrics == "" {
			bestMatch = r // Prefer synced lyrics
		} else if lc.isBetterMatch(r, bestMatch, title, artist) {
			bestMatch = r
		}
	}

	if bestMatch == nil {
		return nil, fmt.Errorf("no suitable lyrics found for %s - %s", artist, title)
	}

	return lc.convertResponse(bestMatch), nil
}

// isBetterMatch checks if candidate is a better match than current
func (lc *LyricsClient) isBetterMatch(candidate, current *lrclibResponse, title, artist string) bool {
	// Simple string matching score
	candidateScore := 0
	currentScore := 0

	titleLower := strings.ToLower(title)
	artistLower := strings.ToLower(artist)

	if strings.Contains(strings.ToLower(candidate.TrackName), titleLower) {
		candidateScore++
	}
	if strings.Contains(strings.ToLower(candidate.ArtistName), artistLower) {
		candidateScore++
	}

	if strings.Contains(strings.ToLower(current.TrackName), titleLower) {
		currentScore++
	}
	if strings.Contains(strings.ToLower(current.ArtistName), artistLower) {
		currentScore++
	}

	return candidateScore > currentScore
}

// convertResponse converts API response to Lyrics struct
func (lc *LyricsClient) convertResponse(r *lrclibResponse) *Lyrics {
	lyrics := &Lyrics{
		Plain:      r.PlainLyrics,
		Synced:     r.SyncedLyrics,
		Source:     "lrclib",
		HasSynced:  r.SyncedLyrics != "",
		TrackName:  r.TrackName,
		ArtistName: r.ArtistName,
		AlbumName:  r.AlbumName,
		Duration:   r.Duration,
	}

	// If only synced lyrics available, extract plain version
	if lyrics.Plain == "" && lyrics.Synced != "" {
		lyrics.Plain = lc.syncedToPlain(lyrics.Synced)
	}

	return lyrics
}

// syncedToPlain converts synced (LRC) lyrics to plain text
func (lc *LyricsClient) syncedToPlain(synced string) string {
	var lines []string
	for _, line := range strings.Split(synced, "\n") {
		// Remove timestamp [mm:ss.xx]
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Find the closing bracket of timestamp
		if strings.HasPrefix(line, "[") {
			idx := strings.Index(line, "]")
			if idx != -1 {
				line = strings.TrimSpace(line[idx+1:])
			}
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

// FetchLyricsForFile fetches lyrics for a file based on its metadata
func (lc *LyricsClient) FetchLyricsForFile(meta *FLACMetadata) (*Lyrics, error) {
	if meta.Title == "" || meta.Artist == "" {
		return nil, fmt.Errorf("missing title or artist metadata")
	}

	return lc.SearchLyrics(meta.Title, meta.Artist, meta.Duration)
}
