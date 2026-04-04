package backend

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	qobuzAPIBase           = "https://www.qobuz.com/api.json/0.2"
	qobuzEndpointBlacklist = 5 * time.Minute
)

// defaultQobuzEndpoints is the built-in list of Qobuz API base URLs.
// Users can extend this via QobuzEndpoints in config.
var defaultQobuzEndpoints = []string{
	"https://www.qobuz.com/api.json/0.2",
}

// qobuzEndpointEntry tracks availability of a single Qobuz API endpoint.
type qobuzEndpointEntry struct {
	url         string
	blacklisted bool
	blacklistAt time.Time
}

// QobuzSource implements MusicSource interface for Qobuz
type QobuzSource struct {
	client        *http.Client
	appID         string
	appSecret     string
	userAuthToken string
	available     bool
	endpoints     []*qobuzEndpointEntry
	endpointMu    sync.Mutex
	logger        *LogBuffer // optional
}

// Qobuz URL patterns
var (
	qobuzTrackRegex    = regexp.MustCompile(`qobuz\.com/[a-z]{2}-[a-z]{2}/track/(\d+)`)
	qobuzAlbumRegex    = regexp.MustCompile(`qobuz\.com/[a-z]{2}-[a-z]{2}/album/[^/]+/([a-z0-9]+)`)
	qobuzPlaylistRegex = regexp.MustCompile(`qobuz\.com/[a-z]{2}-[a-z]{2}/playlist/(\d+)`)
)

// Qobuz API response types
type qobuzTrackResponse struct {
	ID              int    `json:"id"`
	Title           string `json:"title"`
	Duration        int    `json:"duration"`
	TrackNumber     int    `json:"track_number"`
	MediaNumber     int    `json:"media_number"`
	ISRC            string `json:"isrc"`
	ParentalWarning bool   `json:"parental_warning"`
	Performer       struct {
		Name string `json:"name"`
	} `json:"performer"`
	Performers string `json:"performers"`
	Album      struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Artist struct {
			Name string `json:"name"`
		} `json:"artist"`
		Image struct {
			Large string `json:"large"`
			Small string `json:"small"`
		} `json:"image"`
		ReleaseDateOriginal string `json:"release_date_original"`
		Genre               struct {
			Name string `json:"name"`
		} `json:"genre"`
	} `json:"album"`
	Streamable          bool    `json:"streamable"`
	HiresStreamable     bool    `json:"hires_streamable"`
	MaximumBitDepth     int     `json:"maximum_bit_depth"`
	MaximumSamplingRate float64 `json:"maximum_sampling_rate"`
}

type qobuzAlbumResponse struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist struct {
		Name string `json:"name"`
	} `json:"artist"`
	Image struct {
		Large string `json:"large"`
		Small string `json:"small"`
	} `json:"image"`
	ReleaseDateOriginal string `json:"release_date_original"`
	Genre               struct {
		Name string `json:"name"`
	} `json:"genre"`
	TracksCount int `json:"tracks_count"`
	Tracks      struct {
		Items []qobuzTrackResponse `json:"items"`
	} `json:"tracks"`
	Description string `json:"description"`
}

type qobuzPlaylistResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Owner       struct {
		Name string `json:"name"`
	} `json:"owner"`
	Images300   []string `json:"images300"`
	TracksCount int      `json:"tracks_count"`
	Tracks      struct {
		Items []qobuzTrackResponse `json:"items"`
	} `json:"tracks"`
}

type qobuzFileURLResponse struct {
	URL          string  `json:"url"`
	FormatID     int     `json:"format_id"`
	MimeType     string  `json:"mime_type"`
	SamplingRate float64 `json:"sampling_rate"`
	BitDepth     int     `json:"bit_depth"`
}

// NewQobuzSource creates a new Qobuz source
func NewQobuzSource(appID, appSecret string) *QobuzSource {
	q := &QobuzSource{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		appID:     appID,
		appSecret: appSecret,
		available: appID != "" && appSecret != "",
	}
	q.initEndpoints(nil)
	return q
}

// SetLogger attaches a log buffer so endpoint rotation events appear in the Terminal page.
func (q *QobuzSource) SetLogger(logger *LogBuffer) {
	q.logger = logger
}

// SetEndpoints replaces the endpoint pool with a custom list.
// An empty/nil slice reverts to the built-in defaults.
func (q *QobuzSource) SetEndpoints(urls []string) {
	q.initEndpoints(urls)
}

// SetProxy configures the Qobuz HTTP client to use the given proxy URL.
// Supported schemes: http://, https://, socks5://.
// Pass an empty string to remove the proxy.
func (q *QobuzSource) SetProxy(proxyURLStr string) error {
	transport, err := BuildProxyTransport(proxyURLStr)
	if err != nil {
		return err
	}
	q.client = &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
	return nil
}

// initEndpoints (re)initialises the endpoint pool.
func (q *QobuzSource) initEndpoints(urls []string) {
	if len(urls) == 0 {
		urls = defaultQobuzEndpoints
	}
	q.endpointMu.Lock()
	defer q.endpointMu.Unlock()
	q.endpoints = make([]*qobuzEndpointEntry, len(urls))
	for i, u := range urls {
		q.endpoints[i] = &qobuzEndpointEntry{url: u}
	}
}

// getOrderedQobuzEndpoints returns endpoint URLs to try: non-blacklisted first,
// then expired blacklists. Must be called with endpointMu held.
func (q *QobuzSource) getOrderedQobuzEndpoints() []string {
	now := time.Now()
	active := []string{}
	expired := []string{}
	for _, ep := range q.endpoints {
		if !ep.blacklisted {
			active = append(active, ep.url)
		} else if now.After(ep.blacklistAt.Add(qobuzEndpointBlacklist)) {
			ep.blacklisted = false
			active = append(active, ep.url)
		} else {
			expired = append(expired, ep.url)
		}
	}
	return append(active, expired...)
}

// blacklistQobuzEndpoint marks an endpoint as temporarily unavailable.
func (q *QobuzSource) blacklistQobuzEndpoint(rawURL string) {
	q.endpointMu.Lock()
	defer q.endpointMu.Unlock()
	for _, ep := range q.endpoints {
		if ep.url == rawURL {
			ep.blacklisted = true
			ep.blacklistAt = time.Now()
			return
		}
	}
}

// tryQobuzRequest performs a single GET request against one endpoint.
func (q *QobuzSource) tryQobuzRequest(baseURL, endpoint string, params url.Values) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("app_id", q.appID)

	reqURL := fmt.Sprintf("%s/%s?%s", baseURL, endpoint, params.Encode())
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if q.userAuthToken != "" {
		req.Header.Set("X-User-Auth-Token", q.userAuthToken)
	}

	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Qobuz API error %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// makeQobuzRequest tries each endpoint in rotation until one succeeds.
func (q *QobuzSource) makeQobuzRequest(endpoint string, params url.Values) ([]byte, error) {
	q.endpointMu.Lock()
	toTry := q.getOrderedQobuzEndpoints()
	q.endpointMu.Unlock()

	if len(toTry) == 0 {
		toTry = []string{qobuzAPIBase}
	}

	var lastErr error
	var prevBase string

	for _, base := range toTry {
		body, err := q.tryQobuzRequest(base, endpoint, params)
		if err == nil {
			return body, nil
		}

		// Only rotate on server-side errors (5xx / connection failures)
		// 4xx errors (auth, not found) are not endpoint problems
		if isEndpointError(err) {
			lastErr = err
			q.blacklistQobuzEndpoint(base)
			if q.logger != nil {
				if prevBase != "" {
					q.logger.Warn(fmt.Sprintf("switching Qobuz endpoint: %s → %s (reason: %v)", prevBase, base, err))
				} else {
					q.logger.Warn(fmt.Sprintf("Qobuz endpoint %s failed: %v", base, err))
				}
			}
			prevBase = base
			continue
		}
		// Non-endpoint error (auth/not-found): return immediately
		return nil, err
	}

	return nil, fmt.Errorf("all Qobuz endpoints failed: %v", lastErr)
}

// isEndpointError returns true when the error is a network/server error
// that justifies trying a different endpoint.
func isEndpointError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "server returned 5") ||
		strings.Contains(msg, "request failed") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host")
}

// Name returns the source identifier
func (q *QobuzSource) Name() string {
	return "qobuz"
}

// DisplayName returns human-readable name
func (q *QobuzSource) DisplayName() string {
	return "Qobuz"
}

// IsAvailable checks if the source is configured
func (q *QobuzSource) IsAvailable() bool {
	return q.available && q.appID != ""
}

// SetCredentials updates Qobuz credentials
func (q *QobuzSource) SetCredentials(appID, appSecret, userAuthToken string) {
	q.appID = appID
	q.appSecret = appSecret
	q.userAuthToken = userAuthToken
	q.available = appID != "" && appSecret != ""
}

// ParseURL extracts content ID and type from a Qobuz URL
func (q *QobuzSource) ParseURL(rawURL string) (id string, contentType string, err error) {
	if matches := qobuzTrackRegex.FindStringSubmatch(rawURL); len(matches) > 1 {
		return matches[1], "track", nil
	}
	if matches := qobuzAlbumRegex.FindStringSubmatch(rawURL); len(matches) > 1 {
		return matches[1], "album", nil
	}
	if matches := qobuzPlaylistRegex.FindStringSubmatch(rawURL); len(matches) > 1 {
		return matches[1], "playlist", nil
	}
	return "", "", fmt.Errorf("invalid Qobuz URL format")
}

// CanHandleURL checks if this source can handle the given URL
func (q *QobuzSource) CanHandleURL(rawURL string) bool {
	_, _, err := q.ParseURL(rawURL)
	return err == nil
}

// makeRequest performs an authenticated API request with endpoint rotation.
func (q *QobuzSource) makeRequest(endpoint string, params url.Values) ([]byte, error) {
	return q.makeQobuzRequest(endpoint, params)
}

// GetTrack fetches track information by ID
func (q *QobuzSource) GetTrack(id string) (*SourceTrack, error) {
	params := url.Values{}
	params.Set("track_id", id)

	body, err := q.makeRequest("track/get", params)
	if err != nil {
		return nil, err
	}

	var track qobuzTrackResponse
	if err := json.Unmarshal(body, &track); err != nil {
		return nil, fmt.Errorf("failed to parse track: %w", err)
	}

	return q.convertTrack(&track), nil
}

// convertTrack converts Qobuz track to SourceTrack
func (q *QobuzSource) convertTrack(track *qobuzTrackResponse) *SourceTrack {
	artists := []string{track.Performer.Name}
	if track.Performers != "" {
		// Split additional performers
		parts := strings.Split(track.Performers, " - ")
		for _, p := range parts {
			if p != track.Performer.Name {
				artists = append(artists, strings.TrimSpace(p))
			}
		}
	}

	quality := "LOSSLESS"
	if track.HiresStreamable {
		quality = "HI_RES"
	}

	year := ""
	if track.Album.ReleaseDateOriginal != "" && len(track.Album.ReleaseDateOriginal) >= 4 {
		year = track.Album.ReleaseDateOriginal[:4]
	}

	return &SourceTrack{
		ID:          strconv.Itoa(track.ID),
		Title:       track.Title,
		Artist:      track.Performer.Name,
		Artists:     artists,
		Album:       track.Album.Title,
		AlbumID:     track.Album.ID,
		ISRC:        track.ISRC,
		Duration:    track.Duration,
		TrackNumber: track.TrackNumber,
		DiscNumber:  track.MediaNumber,
		Year:        year,
		Genre:       track.Album.Genre.Name,
		CoverURL:    track.Album.Image.Large,
		Explicit:    track.ParentalWarning,
		SourceURL:   fmt.Sprintf("https://play.qobuz.com/track/%d", track.ID),
		Source:      "qobuz",
		Quality:     quality,
	}
}

// GetAlbum fetches album information with tracks
func (q *QobuzSource) GetAlbum(id string) (*SourceAlbum, error) {
	params := url.Values{}
	params.Set("album_id", id)

	body, err := q.makeRequest("album/get", params)
	if err != nil {
		return nil, err
	}

	var album qobuzAlbumResponse
	if err := json.Unmarshal(body, &album); err != nil {
		return nil, fmt.Errorf("failed to parse album: %w", err)
	}

	tracks := make([]SourceTrack, len(album.Tracks.Items))
	for i, t := range album.Tracks.Items {
		tracks[i] = *q.convertTrack(&t)
	}

	year := ""
	if album.ReleaseDateOriginal != "" && len(album.ReleaseDateOriginal) >= 4 {
		year = album.ReleaseDateOriginal[:4]
	}

	return &SourceAlbum{
		ID:          album.ID,
		Title:       album.Title,
		Artist:      album.Artist.Name,
		Year:        year,
		Genre:       album.Genre.Name,
		CoverURL:    album.Image.Large,
		TrackCount:  album.TracksCount,
		Tracks:      tracks,
		Source:      "qobuz",
		SourceURL:   fmt.Sprintf("https://play.qobuz.com/album/%s", album.ID),
		Description: album.Description,
	}, nil
}

// GetPlaylist fetches playlist information with tracks
func (q *QobuzSource) GetPlaylist(id string) (*SourcePlaylist, error) {
	params := url.Values{}
	params.Set("playlist_id", id)
	params.Set("extra", "tracks")

	body, err := q.makeRequest("playlist/get", params)
	if err != nil {
		return nil, err
	}

	var playlist qobuzPlaylistResponse
	if err := json.Unmarshal(body, &playlist); err != nil {
		return nil, fmt.Errorf("failed to parse playlist: %w", err)
	}

	tracks := make([]SourceTrack, len(playlist.Tracks.Items))
	for i, t := range playlist.Tracks.Items {
		tracks[i] = *q.convertTrack(&t)
	}

	coverURL := ""
	if len(playlist.Images300) > 0 {
		coverURL = playlist.Images300[0]
	}

	return &SourcePlaylist{
		ID:          strconv.Itoa(playlist.ID),
		Title:       playlist.Name,
		Description: playlist.Description,
		Creator:     playlist.Owner.Name,
		CoverURL:    coverURL,
		TrackCount:  playlist.TracksCount,
		Tracks:      tracks,
		Source:      "qobuz",
		SourceURL:   fmt.Sprintf("https://play.qobuz.com/playlist/%d", playlist.ID),
	}, nil
}

// GetStreamURL gets the download URL for a track
func (q *QobuzSource) GetStreamURL(trackID string, quality string) (string, error) {
	if q.userAuthToken == "" {
		return "", fmt.Errorf("Qobuz user authentication required for streaming")
	}

	// Format ID: 27 = FLAC 24-bit up to 192kHz, 7 = FLAC 16-bit 44.1kHz, 6 = 320kbps MP3
	formatID := "27"
	if quality == "LOSSLESS" || quality == "CD" {
		formatID = "7"
	}

	// Generate request signature
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signatureInput := fmt.Sprintf("trackgetFileUrlformat_id%sintent_idstreamtrack_id%s%s%s",
		formatID, trackID, timestamp, q.appSecret)
	hash := md5.Sum([]byte(signatureInput))
	signature := hex.EncodeToString(hash[:])

	params := url.Values{}
	params.Set("track_id", trackID)
	params.Set("format_id", formatID)
	params.Set("intent", "stream")
	params.Set("request_ts", timestamp)
	params.Set("request_sig", signature)

	body, err := q.makeRequest("track/getFileUrl", params)
	if err != nil {
		return "", err
	}

	var fileURL qobuzFileURLResponse
	if err := json.Unmarshal(body, &fileURL); err != nil {
		return "", fmt.Errorf("failed to parse file URL: %w", err)
	}

	if fileURL.URL == "" {
		return "", fmt.Errorf("no stream URL returned")
	}

	return fileURL.URL, nil
}

// DownloadTrack downloads a track to the specified directory
func (q *QobuzSource) DownloadTrack(trackID string, outputDir string, options DownloadOptions) (*DownloadResult, error) {
	// Get track info
	track, err := q.GetTrack(trackID)
	if err != nil {
		return nil, fmt.Errorf("failed to get track info: %w", err)
	}

	// Get stream URL
	streamURL, err := q.GetStreamURL(trackID, options.Quality)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream URL: %w", err)
	}

	// Build filename
	filename := buildFilename(options.FileNameFormat, track.Artist, track.Title, track.Album, track.TrackNumber)
	filepath := fmt.Sprintf("%s/%s.flac", outputDir, filename)

	// Download file
	resp, err := q.client.Get(streamURL)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qobuz stream request failed: HTTP %d", resp.StatusCode)
	}

	// Create output file
	file, err := createFile(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Copy data
	size, err := io.Copy(file, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Tag the file
	tagger := NewFLACTagger()
	meta := TrackMetadata{
		Title:       track.Title,
		Artist:      track.Artist,
		Album:       track.Album,
		TrackNumber: track.TrackNumber,
		Year:        track.Year,
		Genre:       track.Genre,
		ISRC:        track.ISRC,
		CoverURL:    track.CoverURL,
	}

	if options.EmbedCover || track.CoverURL != "" {
		if err := tagger.TagFile(filepath, meta); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: failed to tag file: %v\n", err)
		}
	}

	return &DownloadResult{
		TrackID:  track.TrackNumber,
		Title:    track.Title,
		Artist:   track.Artist,
		Album:    track.Album,
		FilePath: filepath,
		FileSize: size,
		Quality:  track.Quality,
		CoverURL: track.CoverURL,
		Success:  true,
	}, nil
}

// qobuzSearchResponse wraps the catalog/search tracks result.
type qobuzSearchResponse struct {
	Tracks struct {
		Items []qobuzTrackResponse `json:"items"`
	} `json:"tracks"`
}

// SearchTrackByISRC searches Qobuz for a track matching the given ISRC.
func (q *QobuzSource) SearchTrackByISRC(isrc string) (*SourceTrack, error) {
	if isrc == "" {
		return nil, fmt.Errorf("ISRC is empty")
	}
	params := url.Values{}
	params.Set("query", isrc)
	params.Set("type", "tracks")
	params.Set("limit", "5")

	body, err := q.makeRequest("catalog/search", params)
	if err != nil {
		return nil, err
	}

	var result qobuzSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	for _, item := range result.Tracks.Items {
		if strings.EqualFold(item.ISRC, isrc) {
			return q.convertTrack(&item), nil
		}
	}
	if len(result.Tracks.Items) > 0 {
		return q.convertTrack(&result.Tracks.Items[0]), nil
	}
	return nil, fmt.Errorf("no Qobuz track found for ISRC %s", isrc)
}

// SearchTrackByTitleArtist searches Qobuz by title and artist name.
func (q *QobuzSource) SearchTrackByTitleArtist(title, artist string) (*SourceTrack, error) {
	query := title
	if artist != "" {
		query = artist + " " + title
	}
	params := url.Values{}
	params.Set("query", query)
	params.Set("type", "tracks")
	params.Set("limit", "5")

	body, err := q.makeRequest("catalog/search", params)
	if err != nil {
		return nil, err
	}

	var result qobuzSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	if len(result.Tracks.Items) == 0 {
		return nil, fmt.Errorf("no Qobuz track found for '%s - %s'", artist, title)
	}
	return q.convertTrack(&result.Tracks.Items[0]), nil
}

// buildFilename creates a filename from template
func buildFilename(format, artist, title, album string, trackNum int) string {
	result := format
	result = strings.ReplaceAll(result, "{artist}", sanitizeFilename(artist))
	result = strings.ReplaceAll(result, "{title}", sanitizeFilename(title))
	result = strings.ReplaceAll(result, "{album}", sanitizeFilename(album))
	result = strings.ReplaceAll(result, "{track}", fmt.Sprintf("%02d", trackNum))
	return result
}

// sanitizeFilename removes invalid characters from filename
func sanitizeFilename(name string) string {
	// Remove/replace invalid characters
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	return strings.TrimSpace(result)
}

// createFile creates a file and necessary directories
func createFile(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	return os.Create(path)
}
