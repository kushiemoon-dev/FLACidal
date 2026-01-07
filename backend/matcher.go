package backend

import (
	"fmt"
	"strings"
	"unicode"
)

// MatchResult represents the result of matching a Tidal track to Spotify
type MatchResult struct {
	TidalTrack   TidalTrack    `json:"tidalTrack"`
	SpotifyTrack *SpotifyTrack `json:"spotifyTrack,omitempty"`
	Matched      bool          `json:"matched"`
	MatchMethod  string        `json:"matchMethod"` // "isrc", "search", "none"
	Confidence   int           `json:"confidence"`  // 0-100
	Error        string        `json:"error,omitempty"`
}

// Matcher handles track matching between services
type Matcher struct {
	spotify *SpotifyClient
	db      *Database
}

// NewMatcher creates a new matcher
func NewMatcher(spotify *SpotifyClient, db *Database) *Matcher {
	return &Matcher{
		spotify: spotify,
		db:      db,
	}
}

// MatchTrack attempts to find a Spotify track matching the Tidal track
func (m *Matcher) MatchTrack(track TidalTrack) MatchResult {
	result := MatchResult{
		TidalTrack: track,
		Matched:    false,
		MatchMethod: "none",
		Confidence: 0,
	}

	// Check cache first
	if m.db != nil {
		if cached, err := m.db.GetCachedTrack(track.ISRC); err == nil && cached != nil {
			result.SpotifyTrack = &SpotifyTrack{
				ID:      cached.SpotifyTrackID,
				URI:     cached.SpotifyURI,
				Name:    cached.Title,
				Artists: cached.Artist,
			}
			result.Matched = true
			result.MatchMethod = cached.MatchMethod
			result.Confidence = int(cached.Confidence)
			return result
		}
	}

	// Try ISRC match first (most reliable)
	if track.ISRC != "" && m.spotify != nil {
		spotifyTrack, err := m.spotify.SearchByISRC(track.ISRC)
		if err != nil {
			result.Error = err.Error()
		} else if spotifyTrack != nil {
			result.SpotifyTrack = spotifyTrack
			result.Matched = true
			result.MatchMethod = "isrc"
			result.Confidence = 100

			// Cache the result
			if m.db != nil {
				m.db.CacheTrack(&CachedTrack{
					ISRC:           track.ISRC,
					TidalTrackID:   fmt.Sprintf("%d", track.ID),
					SpotifyTrackID: spotifyTrack.ID,
					SpotifyURI:     spotifyTrack.URI,
					Title:          track.Title,
					Artist:         track.Artists,
					MatchMethod:    "isrc",
					Confidence:     100,
				})
			}
			return result
		}
	}

	// Fallback to text search
	if m.spotify != nil {
		query := buildSearchQuery(track.Title, track.Artist)
		tracks, err := m.spotify.SearchByQuery(query, 5)
		if err != nil {
			result.Error = err.Error()
		} else if len(tracks) > 0 {
			// Find best match using fuzzy comparison
			bestMatch, confidence := findBestMatch(track, tracks)
			if bestMatch != nil && confidence >= 70 {
				result.SpotifyTrack = bestMatch
				result.Matched = true
				result.MatchMethod = "search"
				result.Confidence = confidence

				// Cache the result
				if m.db != nil && track.ISRC != "" {
					m.db.CacheTrack(&CachedTrack{
						ISRC:           track.ISRC,
						TidalTrackID:   fmt.Sprintf("%d", track.ID),
						SpotifyTrackID: bestMatch.ID,
						SpotifyURI:     bestMatch.URI,
						Title:          track.Title,
						Artist:         track.Artists,
						MatchMethod:    "search",
						Confidence:     float64(confidence),
					})
				}
			}
		}
	}

	return result
}

// MatchPlaylist matches all tracks in a playlist
func (m *Matcher) MatchPlaylist(tracks []TidalTrack) []MatchResult {
	results := make([]MatchResult, len(tracks))
	for i, track := range tracks {
		results[i] = m.MatchTrack(track)
	}
	return results
}

// buildSearchQuery creates a search query from track info
func buildSearchQuery(title, artist string) string {
	// Clean up title (remove featuring, remix info in parentheses for better matches)
	cleanTitle := cleanTrackTitle(title)

	// Get primary artist
	primaryArtist := artist
	if idx := strings.Index(artist, ","); idx > 0 {
		primaryArtist = strings.TrimSpace(artist[:idx])
	}

	return fmt.Sprintf("track:%s artist:%s", cleanTitle, primaryArtist)
}

// cleanTrackTitle removes common suffixes that might interfere with matching
func cleanTrackTitle(title string) string {
	// Remove content in parentheses like "(feat. X)" or "(Remix)"
	result := title

	// Simple parentheses removal
	if idx := strings.Index(result, "("); idx > 0 {
		result = strings.TrimSpace(result[:idx])
	}

	// Remove brackets too
	if idx := strings.Index(result, "["); idx > 0 {
		result = strings.TrimSpace(result[:idx])
	}

	return result
}

// findBestMatch finds the best matching Spotify track from search results
func findBestMatch(tidalTrack TidalTrack, spotifyTracks []SpotifyTrack) (*SpotifyTrack, int) {
	var bestMatch *SpotifyTrack
	bestScore := 0

	tidalTitle := normalize(tidalTrack.Title)
	tidalArtist := normalize(tidalTrack.Artist)

	for i := range spotifyTracks {
		track := &spotifyTracks[i]

		spotifyTitle := normalize(track.Name)
		spotifyArtist := normalize(track.Artists)

		// Calculate similarity scores
		titleSim := similarity(tidalTitle, spotifyTitle)
		artistSim := similarity(tidalArtist, spotifyArtist)

		// Weighted score (title more important)
		score := int(titleSim*0.6 + artistSim*0.4)

		// Bonus for ISRC match
		if track.ISRC != "" && track.ISRC == tidalTrack.ISRC {
			score = 100
		}

		if score > bestScore {
			bestScore = score
			bestMatch = track
		}
	}

	return bestMatch, bestScore
}

// normalize prepares a string for comparison
func normalize(s string) string {
	s = strings.ToLower(s)

	// Remove common words and punctuation
	s = strings.ReplaceAll(s, "&", "and")

	// Keep only alphanumeric and spaces
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			result.WriteRune(r)
		}
	}

	// Collapse multiple spaces
	return strings.Join(strings.Fields(result.String()), " ")
}

// similarity calculates string similarity (0-100)
func similarity(a, b string) float64 {
	if a == b {
		return 100
	}
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	// Use Levenshtein distance
	distance := levenshtein(a, b)
	maxLen := max(len(a), len(b))

	return (1 - float64(distance)/float64(maxLen)) * 100
}

// levenshtein calculates the Levenshtein distance between two strings
func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create matrix
	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(a)][len(b)]
}

func min(nums ...int) int {
	m := nums[0]
	for _, n := range nums[1:] {
		if n < m {
			m = n
		}
	}
	return m
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
