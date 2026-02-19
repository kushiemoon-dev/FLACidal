package backend

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Tidal API integration using Client Credentials flow
// Allows reading public playlists without user login

const (
	tidalAuthURL = "https://auth.tidal.com/v1/oauth2/token"
	tidalAPIBase = "https://api.tidalhifi.com/v1"

	// Internal credentials (same approach as Tidal-Media-Downloader)
	// These have access to playlist API without premium tier
	internalClientID     = "7m7Ap0JC9j1cOM3n"
	internalClientSecret = "vRAdA108tlvkJpTsGZS8rGZ7xTlbJ0qaZ2K9saEzsgY="
)

// TidalClient handles Tidal API requests
type TidalClient struct {
	clientID     string
	clientSecret string
	accessToken  string
	tokenExpiry  time.Time
	httpClient   *http.Client
	mu           sync.Mutex
}

// TidalTrack represents a track from Tidal
type TidalTrack struct {
	ID         int     `json:"id"`
	Title      string  `json:"title"`
	Artist     string  `json:"artist"`
	Artists    string  `json:"artists"` // All artists joined
	Album      string  `json:"album"`
	AlbumID    int     `json:"albumId"`
	ISRC       string  `json:"isrc"`
	Duration   int     `json:"duration"` // seconds
	TrackNum   int     `json:"trackNumber"`
	CoverURL   string  `json:"coverUrl"`
	Explicit   bool    `json:"explicit"`
	TidalURL   string  `json:"tidalUrl"`
	Available  bool    `json:"available"`  // Whether track is available for streaming
	PreviewURL string  `json:"previewUrl,omitempty"` // ~30s MP3 preview URL
	Copyright  string  `json:"copyright,omitempty"`  // Copyright string from album
	Label      string  `json:"label,omitempty"`      // Record label name
	Popularity int     `json:"popularity,omitempty"` // Popularity score 0-100
}

// TidalPlaylist represents a playlist from Tidal
type TidalPlaylist struct {
	UUID        string       `json:"uuid"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Creator     string       `json:"creator"`
	CoverURL    string       `json:"coverUrl"`
	TrackCount  int          `json:"numberOfTracks"`
	Tracks      []TidalTrack `json:"tracks"`
}

// Tidal URL patterns
var (
	tidalPlaylistRegex = regexp.MustCompile(`tidal\.com/(?:browse/)?playlist/([a-f0-9-]+)`)
	tidalTrackRegex    = regexp.MustCompile(`tidal\.com/(?:browse/)?track/(\d+)`)
	tidalAlbumRegex    = regexp.MustCompile(`tidal\.com/(?:browse/)?album/(\d+)`)
	tidalArtistRegex   = regexp.MustCompile(`tidal\.com/(?:browse/)?artist/(\d+)`)
)

// NewTidalClient creates a new Tidal API client
// Uses internal credentials that have playlist API access
func NewTidalClient(clientID, clientSecret string) *TidalClient {
	// Always use internal credentials for playlist access
	// User-provided credentials don't have the required tier
	return &TidalClient{
		clientID:     internalClientID,
		clientSecret: internalClientSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewTidalClientDefault creates a client with internal credentials
func NewTidalClientDefault() *TidalClient {
	return &TidalClient{
		clientID:     internalClientID,
		clientSecret: internalClientSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ParseTidalURL extracts ID and type from a Tidal URL
func ParseTidalURL(rawURL string) (id string, contentType string, err error) {
	if matches := tidalPlaylistRegex.FindStringSubmatch(rawURL); len(matches) > 1 {
		return matches[1], "playlist", nil
	}
	if matches := tidalTrackRegex.FindStringSubmatch(rawURL); len(matches) > 1 {
		return matches[1], "track", nil
	}
	if matches := tidalAlbumRegex.FindStringSubmatch(rawURL); len(matches) > 1 {
		return matches[1], "album", nil
	}
	if matches := tidalArtistRegex.FindStringSubmatch(rawURL); len(matches) > 1 {
		return matches[1], "artist", nil
	}
	return "", "", fmt.Errorf("invalid Tidal URL: %s", rawURL)
}

// IsTidalPlaylistURL checks if URL is a Tidal playlist URL
func IsTidalPlaylistURL(rawURL string) bool {
	return tidalPlaylistRegex.MatchString(rawURL)
}

// authenticate gets or refreshes the access token
func (c *TidalClient) authenticate() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return if token is still valid (with 60s buffer)
	if c.accessToken != "" && time.Now().Add(60*time.Second).Before(c.tokenExpiry) {
		return nil
	}

	// Request new token
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)

	req, err := http.NewRequest("POST", tidalAuthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("auth failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return nil
}

// doRequest makes an authenticated request to Tidal API
func (c *TidalClient) doRequest(endpoint string) ([]byte, error) {
	if err := c.authenticate(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", tidalAPIBase+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// GetPlaylist fetches a public playlist by UUID
func (c *TidalClient) GetPlaylist(playlistUUID string) (*TidalPlaylist, error) {
	// Fetch playlist metadata
	endpoint := fmt.Sprintf("/playlists/%s?countryCode=US", playlistUUID)
	data, err := c.doRequest(endpoint)
	if err != nil {
		// Check if it's a 404 error - likely a private playlist
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			return nil, fmt.Errorf("playlist not found - it may be private. Only public playlists can be accessed")
		}
		return nil, fmt.Errorf("failed to fetch playlist: %w", err)
	}

	var playlistResp struct {
		UUID           string `json:"uuid"`
		Title          string `json:"title"`
		Description    string `json:"description"`
		NumberOfTracks int    `json:"numberOfTracks"`
		Creator        struct {
			Name string `json:"name"`
		} `json:"creator"`
		Image       string `json:"image"`
		SquareImage string `json:"squareImage"`
	}

	if err := json.Unmarshal(data, &playlistResp); err != nil {
		return nil, fmt.Errorf("failed to parse playlist: %w", err)
	}

	// Use squareImage for playlists (image field often doesn't work)
	coverImage := playlistResp.SquareImage
	if coverImage == "" {
		coverImage = playlistResp.Image
	}

	// Creator name fallback
	creatorName := playlistResp.Creator.Name
	if creatorName == "" {
		creatorName = "Tidal Playlist"
	}

	playlist := &TidalPlaylist{
		UUID:        playlistResp.UUID,
		Title:       playlistResp.Title,
		Description: playlistResp.Description,
		Creator:     creatorName,
		TrackCount:  playlistResp.NumberOfTracks,
		CoverURL:    formatTidalImageURL(coverImage),
	}

	// Fetch all tracks with pagination
	tracks, err := c.getPlaylistTracks(playlistUUID, playlistResp.NumberOfTracks)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tracks: %w", err)
	}
	playlist.Tracks = tracks

	return playlist, nil
}

// getPlaylistTracks fetches all tracks from a playlist with pagination
func (c *TidalClient) getPlaylistTracks(playlistUUID string, totalTracks int) ([]TidalTrack, error) {
	var allTracks []TidalTrack
	limit := 100
	offset := 0

	for offset < totalTracks {
		endpoint := fmt.Sprintf("/playlists/%s/items?countryCode=US&limit=%d&offset=%d",
			playlistUUID, limit, offset)

		data, err := c.doRequest(endpoint)
		if err != nil {
			return nil, err
		}

		var itemsResp struct {
			Items []struct {
				Item struct {
					ID              int    `json:"id"`
					Title           string `json:"title"`
					Duration        int    `json:"duration"`
					ISRC            string `json:"isrc"`
					Explicit        bool   `json:"explicit"`
					StreamReady     *bool  `json:"streamReady"`    // nil = not in response = assume available
					AudioPreviewURL string `json:"audioPreviewUrl"`
					Popularity      int    `json:"popularity"`
					Album           struct {
						ID    int    `json:"id"`
						Title string `json:"title"`
						Cover string `json:"cover"`
					} `json:"album"`
					Artists []struct {
						ID   int    `json:"id"`
						Name string `json:"name"`
					} `json:"artists"`
					TrackNumber int `json:"trackNumber"`
				} `json:"item"`
			} `json:"items"`
		}

		if err := json.Unmarshal(data, &itemsResp); err != nil {
			return nil, fmt.Errorf("failed to parse tracks: %w", err)
		}

		for _, item := range itemsResp.Items {
			track := item.Item

			// Build artist string
			var artistNames []string
			for _, a := range track.Artists {
				artistNames = append(artistNames, a.Name)
			}
			artistStr := strings.Join(artistNames, ", ")
			mainArtist := ""
			if len(track.Artists) > 0 {
				mainArtist = track.Artists[0].Name
			}
			// Available = true unless streamReady is explicitly false
			available := track.StreamReady == nil || *track.StreamReady

			allTracks = append(allTracks, TidalTrack{
				ID:         track.ID,
				Title:      track.Title,
				Artist:     mainArtist,
				Artists:    artistStr,
				Album:      track.Album.Title,
				AlbumID:    track.Album.ID,
				ISRC:       track.ISRC,
				Duration:   track.Duration,
				TrackNum:   track.TrackNumber,
				CoverURL:   formatTidalImageURL(track.Album.Cover),
				Explicit:   track.Explicit,
				TidalURL:   fmt.Sprintf("https://tidal.com/browse/track/%d", track.ID),
				Available:  available,
				PreviewURL: track.AudioPreviewURL,
				Popularity: track.Popularity,
			})
		}

		offset += limit
	}

	return allTracks, nil
}

// formatTidalImageURL converts Tidal image ID to full URL
func formatTidalImageURL(imageID string) string {
	if imageID == "" {
		return ""
	}
	// Replace dashes with slashes for Tidal image URL format
	imageID = strings.ReplaceAll(imageID, "-", "/")
	return fmt.Sprintf("https://resources.tidal.com/images/%s/640x640.jpg", imageID)
}

// GetTrack fetches a single track by ID
func (c *TidalClient) GetTrack(trackID string) (*TidalTrack, error) {
	endpoint := fmt.Sprintf("/tracks/%s?countryCode=US", trackID)
	data, err := c.doRequest(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch track: %w", err)
	}

	var trackResp struct {
		ID              int    `json:"id"`
		Title           string `json:"title"`
		Duration        int    `json:"duration"`
		ISRC            string `json:"isrc"`
		Explicit        bool   `json:"explicit"`
		StreamReady     *bool  `json:"streamReady"`
		AudioPreviewURL string `json:"audioPreviewUrl"`
		Popularity      int    `json:"popularity"`
		Album           struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
			Cover string `json:"cover"`
		} `json:"album"`
		Artists []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"artists"`
		TrackNumber int `json:"trackNumber"`
	}

	if err := json.Unmarshal(data, &trackResp); err != nil {
		return nil, fmt.Errorf("failed to parse track: %w", err)
	}

	var artistNames []string
	for _, a := range trackResp.Artists {
		artistNames = append(artistNames, a.Name)
	}
	artistStr := strings.Join(artistNames, ", ")
	mainArtist := ""
	if len(trackResp.Artists) > 0 {
		mainArtist = trackResp.Artists[0].Name
	}

	trackAvailable := trackResp.StreamReady == nil || *trackResp.StreamReady
	return &TidalTrack{
		ID:         trackResp.ID,
		Title:      trackResp.Title,
		Artist:     mainArtist,
		Artists:    artistStr,
		Album:      trackResp.Album.Title,
		AlbumID:    trackResp.Album.ID,
		ISRC:       trackResp.ISRC,
		Duration:   trackResp.Duration,
		TrackNum:   trackResp.TrackNumber,
		CoverURL:   formatTidalImageURL(trackResp.Album.Cover),
		Explicit:   trackResp.Explicit,
		TidalURL:   fmt.Sprintf("https://tidal.com/browse/track/%d", trackResp.ID),
		Available:  trackAvailable,
		PreviewURL: trackResp.AudioPreviewURL,
		Popularity: trackResp.Popularity,
	}, nil
}

// TidalAlbum represents album info with tracks
type TidalAlbum struct {
	ID          int          `json:"id"`
	Title       string       `json:"title"`
	Artist      string       `json:"artist"`
	ReleaseDate string       `json:"releaseDate"`
	TrackCount  int          `json:"trackCount"`
	CoverURL    string       `json:"coverUrl"`
	AlbumType   string       `json:"albumType,omitempty"` // "ALBUM", "EP", "SINGLE", "COMPILATION"
	Copyright   string       `json:"copyright,omitempty"` // Copyright string from Tidal
	Label       string       `json:"label,omitempty"`     // Record label name
	Tracks      []TidalTrack `json:"tracks"`
}

// TidalArtist represents an artist with their discography
type TidalArtist struct {
	ID         int          `json:"id"`
	Name       string       `json:"name"`
	PictureURL string       `json:"pictureUrl,omitempty"`
	Albums     []TidalAlbum `json:"albums"`
}

// GetAlbum fetches an album with all its tracks
func (c *TidalClient) GetAlbum(albumID string) (*TidalAlbum, error) {
	// Fetch album metadata
	endpoint := fmt.Sprintf("/albums/%s?countryCode=US", albumID)
	data, err := c.doRequest(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch album: %w", err)
	}

	var albumResp struct {
		ID             int    `json:"id"`
		Title          string `json:"title"`
		Type           string `json:"type"`
		ReleaseDate    string `json:"releaseDate"`
		NumberOfTracks int    `json:"numberOfTracks"`
		Cover          string `json:"cover"`
		Copyright      string `json:"copyright"`
		Artists        []struct {
			Name string `json:"name"`
		} `json:"artists"`
		Label struct {
			Name string `json:"name"`
		} `json:"label"`
	}

	if err := json.Unmarshal(data, &albumResp); err != nil {
		return nil, fmt.Errorf("failed to parse album: %w", err)
	}

	artistName := ""
	if len(albumResp.Artists) > 0 {
		artistName = albumResp.Artists[0].Name
	}

	album := &TidalAlbum{
		ID:          albumResp.ID,
		Title:       albumResp.Title,
		Artist:      artistName,
		ReleaseDate: albumResp.ReleaseDate,
		TrackCount:  albumResp.NumberOfTracks,
		CoverURL:    formatTidalImageURL(albumResp.Cover),
		AlbumType:   albumResp.Type,
		Copyright:   albumResp.Copyright,
		Label:       albumResp.Label.Name,
	}

	// Fetch album tracks
	tracksEndpoint := fmt.Sprintf("/albums/%s/tracks?countryCode=US&limit=100", albumID)
	tracksData, err := c.doRequest(tracksEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch album tracks: %w", err)
	}

	var tracksResp struct {
		Items []struct {
			ID              int    `json:"id"`
			Title           string `json:"title"`
			Duration        int    `json:"duration"`
			ISRC            string `json:"isrc"`
			Explicit        bool   `json:"explicit"`
			StreamReady     *bool  `json:"streamReady"`
			AudioPreviewURL string `json:"audioPreviewUrl"`
			Popularity      int    `json:"popularity"`
			Album           struct {
				ID    int    `json:"id"`
				Title string `json:"title"`
				Cover string `json:"cover"`
			} `json:"album"`
			Artists []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"artists"`
			TrackNumber int `json:"trackNumber"`
		} `json:"items"`
	}

	if err := json.Unmarshal(tracksData, &tracksResp); err != nil {
		return nil, fmt.Errorf("failed to parse album tracks: %w", err)
	}

	for _, track := range tracksResp.Items {
		var artistNames []string
		for _, a := range track.Artists {
			artistNames = append(artistNames, a.Name)
		}
		artistStr := strings.Join(artistNames, ", ")
		mainArtist := ""
		if len(track.Artists) > 0 {
			mainArtist = track.Artists[0].Name
		}

		trackAvail := track.StreamReady == nil || *track.StreamReady
		album.Tracks = append(album.Tracks, TidalTrack{
			ID:         track.ID,
			Title:      track.Title,
			Artist:     mainArtist,
			Artists:    artistStr,
			Album:      track.Album.Title,
			AlbumID:    track.Album.ID,
			ISRC:       track.ISRC,
			Duration:   track.Duration,
			TrackNum:   track.TrackNumber,
			CoverURL:   formatTidalImageURL(track.Album.Cover),
			Explicit:   track.Explicit,
			TidalURL:   fmt.Sprintf("https://tidal.com/browse/track/%d", track.ID),
			Available:  trackAvail,
			PreviewURL: track.AudioPreviewURL,
			Copyright:  album.Copyright,
			Label:      album.Label,
			Popularity: track.Popularity,
		})
	}

	return album, nil
}

// GetArtistDiscography fetches an artist's basic info and all their albums (without track lists).
// Callers can fetch individual album tracks with GetAlbum if needed.
func (c *TidalClient) GetArtistDiscography(artistID string) (*TidalArtist, error) {
	// Fetch artist metadata
	artistData, err := c.doRequest(fmt.Sprintf("/artists/%s?countryCode=US", artistID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch artist: %w", err)
	}

	var artistResp struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.Unmarshal(artistData, &artistResp); err != nil {
		return nil, fmt.Errorf("failed to parse artist: %w", err)
	}

	artist := &TidalArtist{
		ID:         artistResp.ID,
		Name:       artistResp.Name,
		PictureURL: formatTidalImageURL(artistResp.Picture),
	}

	// Fetch all albums with pagination (filter=ALL: albums, EPs, singles, compilations)
	var allAlbums []TidalAlbum
	limit := 50
	offset := 0
	for {
		albumsData, err := c.doRequest(fmt.Sprintf(
			"/artists/%s/albums?countryCode=US&limit=%d&offset=%d&filter=ALL",
			artistID, limit, offset,
		))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch artist albums: %w", err)
		}

		var albumsResp struct {
			Items []struct {
				ID             int    `json:"id"`
				Title          string `json:"title"`
				ReleaseDate    string `json:"releaseDate"`
				NumberOfTracks int    `json:"numberOfTracks"`
				Cover          string `json:"cover"`
				Type           string `json:"type"`
				Artists        []struct {
					Name string `json:"name"`
				} `json:"artists"`
			} `json:"items"`
			TotalNumberOfItems int `json:"totalNumberOfItems"`
		}
		if err := json.Unmarshal(albumsData, &albumsResp); err != nil {
			return nil, fmt.Errorf("failed to parse artist albums: %w", err)
		}

		for _, a := range albumsResp.Items {
			artistName := artist.Name
			if len(a.Artists) > 0 {
				artistName = a.Artists[0].Name
			}
			allAlbums = append(allAlbums, TidalAlbum{
				ID:          a.ID,
				Title:       a.Title,
				Artist:      artistName,
				ReleaseDate: a.ReleaseDate,
				TrackCount:  a.NumberOfTracks,
				CoverURL:    formatTidalImageURL(a.Cover),
				AlbumType:   a.Type,
			})
		}

		offset += limit
		if offset >= albumsResp.TotalNumberOfItems || len(albumsResp.Items) == 0 {
			break
		}
	}

	artist.Albums = allAlbums
	return artist, nil
}

// ArtistImageURLs returns CDN URLs for multiple sizes of an artist's picture.
// rawPictureID is the dashed UUID returned by the Tidal API (e.g. "11-22-33-...").
// Returns a map of label → URL: "profile" (640×640), "profile_hires" (1280×1280), "banner" (1080×720).
func ArtistImageURLs(rawPictureID string) map[string]string {
	if rawPictureID == "" {
		return nil
	}
	formatted := strings.ReplaceAll(rawPictureID, "-", "/")
	base := "https://resources.tidal.com/images/" + formatted
	return map[string]string{
		"profile":       base + "/640x640.jpg",
		"profile_hires": base + "/1280x1280.jpg",
		"banner":        base + "/1080x720.jpg",
	}
}

// GetArtistPictureID fetches only the artist name and raw picture ID (no album pagination).
func (c *TidalClient) GetArtistPictureID(artistID string) (name string, pictureID string, err error) {
	data, err := c.doRequest(fmt.Sprintf("/artists/%s?countryCode=US", artistID))
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch artist: %w", err)
	}
	var resp struct {
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", "", fmt.Errorf("failed to parse artist: %w", err)
	}
	return resp.Name, resp.Picture, nil
}
