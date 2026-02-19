package backend

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Tidal HiFi API Service - vogel.qqdl.site
// Free Tidal FLAC proxy without credentials (same as mkv-video)

const (
	tidalHifiAPIBase     = "https://vogel.qqdl.site"
	endpointBlacklistDur = 5 * time.Minute
)

// defaultTidalHifiEndpoints is the built-in list of Tidal HiFi proxy endpoints.
// Users can extend this via TidalHifiEndpoints in config.
var defaultTidalHifiEndpoints = []string{
	"https://vogel.qqdl.site",
}

// endpointEntry tracks availability of a single API endpoint.
type endpointEntry struct {
	url         string
	blacklisted bool
	blacklistAt time.Time
}

// TidalHifiService implements FLAC downloading via a pool of proxy endpoints
type TidalHifiService struct {
	client         *http.Client
	downloadClient *http.Client // Separate client for downloads (no timeout)
	endpoints      []*endpointEntry
	endpointMu     sync.Mutex
	options        DownloadOptions
	logger         *LogBuffer // optional — set via SetLogger
}

// TidalManifest represents the decoded manifest from hifi-api
type TidalManifest struct {
	MimeType       string   `json:"mimeType"`
	Codecs         string   `json:"codecs"`
	EncryptionType string   `json:"encryptionType"`
	URLs           []string `json:"urls"`
}

// TidalHifiTrackResponse represents the track info response from vogel
type TidalHifiTrackResponse struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Duration    int    `json:"duration"`
	TrackNumber int    `json:"trackNumber"`
	ISRC        string `json:"isrc"`
	Explicit    bool   `json:"explicit"`
	Artist      struct {
		Name string `json:"name"`
	} `json:"artist"`
	Artists []struct {
		Name string `json:"name"`
	} `json:"artists"`
	Album struct {
		Title string `json:"title"`
		Cover string `json:"cover"`
	} `json:"album"`
}

// TidalStreamResponse represents the stream/manifest response
type TidalStreamResponse struct {
	TrackID      int    `json:"trackId"`
	AssetID      int    `json:"assetId,omitempty"`
	AudioMode    string `json:"audioMode"`
	AudioQuality string `json:"audioQuality"`
	Manifest     string `json:"manifest"`
	ManifestType string `json:"manifestMimeType"`
}

// TidalInfoResponse wraps track info with version
type TidalInfoResponse struct {
	Version string                 `json:"version"`
	Data    TidalHifiTrackResponse `json:"data"`
}

// TidalStreamDataResponse wraps stream response with version
type TidalStreamDataResponse struct {
	Version string              `json:"version"`
	Data    TidalStreamResponse `json:"data"`
}

// StreamInfo contains information about the audio stream returned by Tidal
type StreamInfo struct {
	URL          string `json:"url"`
	AudioQuality string `json:"audioQuality"` // HI_RES, LOSSLESS, HIGH, etc.
	AudioMode    string `json:"audioMode"`    // STEREO, DOLBY_ATMOS, etc.
}

// DownloadResult represents the result of a download
type DownloadResult struct {
	TrackID         int              `json:"trackId"`
	Title           string           `json:"title"`
	Artist          string           `json:"artist"`
	Album           string           `json:"album"`
	FilePath        string           `json:"filePath"`
	FileSize        int64            `json:"fileSize"`
	Quality         string           `json:"quality"`
	RequestedQuality string          `json:"requestedQuality,omitempty"` // Quality that was requested
	QualityMismatch bool             `json:"qualityMismatch,omitempty"`  // True if server returned different quality
	CoverURL        string           `json:"coverUrl"`
	Success         bool             `json:"success"`
	Error           string           `json:"error,omitempty"`
	Analysis        *AnalysisResult  `json:"analysis,omitempty"` // Auto-analysis result if enabled
}

// DownloadOptions configures download behavior
type DownloadOptions struct {
	Quality         string // "HI_RES", "LOSSLESS", "HIGH"
	FileNameFormat  string // "{artist} - {title}", "{track} - {title}", etc.
	OrganizeFolders bool   // Create Artist/Album/ subfolders
	EmbedCover      bool   // Embed cover art in FLAC
	SaveCoverFile   bool   // Save cover art as .jpg file next to FLAC
	AutoAnalyze     bool   // Automatically analyze quality after download
}

// NewTidalHifiService creates a new Tidal HiFi download service
func NewTidalHifiService() *TidalHifiService {
	// Transport with connection pooling for downloads
	downloadTransport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     90 * time.Second,
	}

	svc := &TidalHifiService{
		client: &http.Client{
			Timeout: 30 * time.Second, // For API calls only
		},
		downloadClient: &http.Client{
			Timeout:   0, // No timeout for downloads
			Transport: downloadTransport,
		},
		options: DownloadOptions{
			Quality:         "LOSSLESS",
			FileNameFormat:  "{artist} - {title}",
			OrganizeFolders: false,
			EmbedCover:      true,
			SaveCoverFile:   true,
		},
	}
	svc.initEndpoints(nil) // uses defaultTidalHifiEndpoints
	return svc
}

// SetLogger attaches a log buffer so endpoint rotation events are visible in the Terminal page.
func (t *TidalHifiService) SetLogger(logger *LogBuffer) {
	t.logger = logger
}

// SetEndpoints replaces the endpoint pool with a custom list (e.g. from config).
// An empty/nil slice reverts to the built-in defaults.
func (t *TidalHifiService) SetEndpoints(urls []string) {
	t.initEndpoints(urls)
}

// initEndpoints (re)initialises the endpoint pool.
func (t *TidalHifiService) initEndpoints(urls []string) {
	if len(urls) == 0 {
		urls = defaultTidalHifiEndpoints
	}
	t.endpointMu.Lock()
	defer t.endpointMu.Unlock()
	t.endpoints = make([]*endpointEntry, len(urls))
	for i, u := range urls {
		t.endpoints[i] = &endpointEntry{url: u}
	}
}

// getOrderedEndpoints returns endpoint URLs to try: non-blacklisted first,
// then expired blacklists. Must be called with endpointMu held.
func (t *TidalHifiService) getOrderedEndpoints() []string {
	now := time.Now()
	active := []string{}
	expired := []string{}
	for _, ep := range t.endpoints {
		if !ep.blacklisted {
			active = append(active, ep.url)
		} else if now.After(ep.blacklistAt.Add(endpointBlacklistDur)) {
			// Blacklist has expired — treat as available again
			ep.blacklisted = false
			active = append(active, ep.url)
		} else {
			expired = append(expired, ep.url)
		}
	}
	// Return active first, then still-blacklisted as final fallback
	return append(active, expired...)
}

// blacklistEndpoint marks an endpoint as temporarily unavailable.
func (t *TidalHifiService) blacklistEndpoint(rawURL string) {
	t.endpointMu.Lock()
	defer t.endpointMu.Unlock()
	for _, ep := range t.endpoints {
		if ep.url == rawURL {
			ep.blacklisted = true
			ep.blacklistAt = time.Now()
			return
		}
	}
}

// selectEndpoint returns the first usable endpoint URL.
func (t *TidalHifiService) selectEndpoint() string {
	t.endpointMu.Lock()
	defer t.endpointMu.Unlock()
	ordered := t.getOrderedEndpoints()
	if len(ordered) > 0 {
		return ordered[0]
	}
	return tidalHifiAPIBase
}

// tryAPIRequest performs a single GET request against one endpoint+path.
func (t *TidalHifiService) tryAPIRequest(endpoint, path string) ([]byte, error) {
	req, err := http.NewRequest("GET", endpoint+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// makeAPIRequest tries each endpoint in rotation until one succeeds.
// Failing endpoints are temporarily blacklisted for endpointBlacklistDur.
func (t *TidalHifiService) makeAPIRequest(path string) ([]byte, error) {
	t.endpointMu.Lock()
	toTry := t.getOrderedEndpoints()
	t.endpointMu.Unlock()

	if len(toTry) == 0 {
		toTry = []string{tidalHifiAPIBase}
	}

	var lastErr error
	var prevEndpoint string

	for _, endpoint := range toTry {
		body, err := t.tryAPIRequest(endpoint, path)
		if err == nil {
			return body, nil
		}

		lastErr = err
		t.blacklistEndpoint(endpoint)

		if t.logger != nil {
			if prevEndpoint != "" {
				t.logger.Warn(fmt.Sprintf("switching Tidal endpoint: %s → %s (reason: %v)", prevEndpoint, endpoint, err))
			} else {
				t.logger.Warn(fmt.Sprintf("Tidal endpoint %s failed: %v", endpoint, err))
			}
		}
		prevEndpoint = endpoint
	}

	return nil, fmt.Errorf("all Tidal endpoints failed: %v", lastErr)
}

// SetOptions updates download options
func (t *TidalHifiService) SetOptions(opts DownloadOptions) {
	t.options = opts
}

// GetOptions returns current download options
func (t *TidalHifiService) GetOptions() DownloadOptions {
	return t.options
}

// IsAvailable checks if the service is reachable
func (t *TidalHifiService) IsAvailable() bool {
	endpoint := t.selectEndpoint()
	resp, err := t.client.Head(endpoint)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}

// GetTrackByID fetches track info by Tidal ID
func (t *TidalHifiService) GetTrackByID(trackID int) (*TidalHifiTrackResponse, error) {
	body, err := t.makeAPIRequest(fmt.Sprintf("/info/?id=%d", trackID))
	if err != nil {
		return nil, fmt.Errorf("info request failed: %w", err)
	}

	// Try v2.0 wrapper format first
	var infoResp TidalInfoResponse
	if err := json.Unmarshal(body, &infoResp); err != nil {
		return nil, fmt.Errorf("failed to parse track info: %w", err)
	}

	// Check if we got data from the wrapper
	if infoResp.Data.ID > 0 {
		return &infoResp.Data, nil
	}

	// Fallback: try direct format
	var trackInfo TidalHifiTrackResponse
	if err := json.Unmarshal(body, &trackInfo); err != nil {
		return nil, fmt.Errorf("failed to parse track info (direct): %w", err)
	}

	return &trackInfo, nil
}

// GetStreamURL fetches the FLAC stream URL for a track and returns stream info
func (t *TidalHifiService) GetStreamURL(trackID int) (*StreamInfo, error) {
	quality := t.options.Quality
	if quality == "" {
		quality = "LOSSLESS"
	}

	body, err := t.makeAPIRequest(fmt.Sprintf("/track/?id=%d&quality=%s", trackID, quality))
	if err != nil {
		return nil, fmt.Errorf("stream request failed: %w", err)
	}

	// Try v2.0 wrapper format first
	var streamDataResp TidalStreamDataResponse
	if err := json.Unmarshal(body, &streamDataResp); err != nil {
		return nil, fmt.Errorf("failed to parse stream response: %w", err)
	}

	// Extract quality info from response
	audioQuality := streamDataResp.Data.AudioQuality
	audioMode := streamDataResp.Data.AudioMode
	manifestBase64 := streamDataResp.Data.Manifest

	if manifestBase64 == "" {
		// Fallback: try direct format
		var streamResp TidalStreamResponse
		if err := json.Unmarshal(body, &streamResp); err != nil {
			return nil, fmt.Errorf("failed to parse stream response (direct): %w", err)
		}
		manifestBase64 = streamResp.Manifest
		audioQuality = streamResp.AudioQuality
		audioMode = streamResp.AudioMode
	}

	if manifestBase64 == "" {
		return nil, fmt.Errorf("no manifest in stream response")
	}

	// Decode base64 manifest
	manifestBytes, err := base64.StdEncoding.DecodeString(manifestBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %w", err)
	}

	var manifest TidalManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if len(manifest.URLs) == 0 {
		return nil, fmt.Errorf("no download URLs in manifest")
	}

	return &StreamInfo{
		URL:          manifest.URLs[0],
		AudioQuality: audioQuality,
		AudioMode:    audioMode,
	}, nil
}

// parseSearchBody extracts track items from a Tidal search response body.
func parseSearchBody(body []byte) ([]TidalHifiTrackResponse, error) {
	var result struct {
		Version string `json:"version,omitempty"`
		Data    struct {
			Items []TidalHifiTrackResponse `json:"items"`
		} `json:"data,omitempty"`
		Tracks struct {
			Items []TidalHifiTrackResponse `json:"items"`
		} `json:"tracks,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}
	if len(result.Data.Items) > 0 {
		return result.Data.Items, nil
	}
	return result.Tracks.Items, nil
}

// SearchTrack searches for a track on Tidal via vogel
func (t *TidalHifiService) SearchTrack(query string) (*TidalHifiTrackResponse, error) {
	body, err := t.makeAPIRequest("/search/?s=" + url.QueryEscape(query))
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	items, err := parseSearchBody(body)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no tracks found for query: %s", query)
	}
	return &items[0], nil
}

// SearchTracks searches for tracks on Tidal via vogel and returns multiple results
func (t *TidalHifiService) SearchTracks(query string, limit int) ([]TidalHifiTrackResponse, error) {
	if limit <= 0 {
		limit = 20
	}

	body, err := t.makeAPIRequest("/search/?s=" + url.QueryEscape(query))
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	items, err := parseSearchBody(body)
	if err != nil {
		return nil, err
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

// DownloadTrack downloads a single track to the specified directory
func (t *TidalHifiService) DownloadTrack(trackID int, outputDir string) (*DownloadResult, error) {
	result := &DownloadResult{
		TrackID: trackID,
		Success: false,
	}

	// Get track info
	track, err := t.GetTrackByID(trackID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get track info: %v", err)
		return result, err
	}

	artistName := track.Artist.Name
	if artistName == "" && len(track.Artists) > 0 {
		artistName = track.Artists[0].Name
	}

	result.Title = track.Title
	result.Artist = artistName
	result.Album = track.Album.Title

	coverURL := ""
	if track.Album.Cover != "" {
		coverURL = fmt.Sprintf("https://resources.tidal.com/images/%s/1280x1280.jpg",
			strings.ReplaceAll(track.Album.Cover, "-", "/"))
		result.CoverURL = coverURL
	}

	// Get stream URL and quality info
	streamInfo, err := t.GetStreamURL(trackID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get stream URL: %v", err)
		return result, err
	}

	// Use the actual quality from the server response
	requestedQuality := t.options.Quality
	if requestedQuality == "" {
		requestedQuality = "LOSSLESS"
	}
	result.RequestedQuality = requestedQuality
	result.Quality = streamInfo.AudioQuality

	// Check for quality mismatch and log warning
	if streamInfo.AudioQuality != "" && streamInfo.AudioQuality != requestedQuality {
		result.QualityMismatch = true
		println(fmt.Sprintf("Warning: requested %s but got %s for track %d (%s - %s)",
			requestedQuality, streamInfo.AudioQuality, trackID, artistName, track.Title))
	}

	// Determine output path based on options
	finalDir := outputDir
	if t.options.OrganizeFolders {
		// Create Artist/Album subfolders
		safeArtist := SanitizeFileName(artistName)
		safeAlbum := SanitizeFileName(track.Album.Title)
		if safeAlbum == "" {
			safeAlbum = "Singles"
		}
		finalDir = filepath.Join(outputDir, safeArtist, safeAlbum)
	}

	// Create output directory
	if err := os.MkdirAll(finalDir, 0755); err != nil {
		result.Error = fmt.Sprintf("failed to create output directory: %v", err)
		return result, err
	}

	// Generate filename based on format
	fileName := t.formatFileName(track, artistName)
	outputPath := filepath.Join(finalDir, fmt.Sprintf("%s.flac", fileName))
	result.FilePath = outputPath

	// Check if file already exists (skip if already downloaded)
	if stat, err := os.Stat(outputPath); err == nil && stat.Size() > 0 {
		result.FileSize = stat.Size()
		result.Success = true
		result.Error = "skipped: already exists"
		return result, nil
	}

	// Download the FLAC file
	if err := t.downloadFile(streamInfo.URL, outputPath); err != nil {
		result.Error = fmt.Sprintf("download failed: %v", err)
		return result, err
	}

	// Tag the file with metadata
	tagger := NewFLACTagger()
	meta := TrackMetadata{
		Title:       track.Title,
		Artist:      artistName,
		Album:       track.Album.Title,
		TrackNumber: track.TrackNumber,
		ISRC:        track.ISRC,
	}

	// Only embed cover if option is enabled
	if t.options.EmbedCover {
		meta.CoverURL = coverURL
	}

	if err := tagger.TagFile(outputPath, meta); err != nil {
		// Log but don't fail - file is still downloaded
		println("Warning: failed to tag file:", err.Error())
	}

	// Save cover as separate .jpg file if enabled
	if t.options.SaveCoverFile && coverURL != "" {
		coverPath := strings.TrimSuffix(outputPath, ".flac") + ".jpg"
		if err := t.saveCoverFile(coverURL, coverPath); err != nil {
			// Log but don't fail - FLAC is still downloaded
			println("Warning: failed to save cover file:", err.Error())
		}
	}

	// Auto-analyze the downloaded file if enabled
	if t.options.AutoAnalyze {
		analysis, err := AnalyzeFLAC(outputPath)
		if err == nil {
			result.Analysis = analysis
			if !analysis.IsTrueLossless {
				println(fmt.Sprintf("Warning: %s may be upscaled from lossy source (verdict: %s)",
					outputPath, analysis.VerdictLabel))
			}
		} else {
			println("Warning: failed to analyze file:", err.Error())
		}
	}

	// Get file size
	stat, _ := os.Stat(outputPath)
	if stat != nil {
		result.FileSize = stat.Size()
	}

	result.Success = true
	return result, nil
}

func (t *TidalHifiService) downloadFile(downloadURL, outputPath string) error {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := t.downloadClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to start download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download server returned %d", resp.StatusCode)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		os.Remove(outputPath) // Clean up partial file
		return fmt.Errorf("download interrupted: %w", err)
	}

	return nil
}

// saveCoverFile downloads and saves cover art as a separate .jpg file
func (t *TidalHifiService) saveCoverFile(coverURL, outputPath string) error {
	// Skip if file already exists
	if _, err := os.Stat(outputPath); err == nil {
		return nil
	}

	resp, err := http.Get(coverURL)
	if err != nil {
		return fmt.Errorf("failed to download cover: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("cover server returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read cover data: %w", err)
	}

	return os.WriteFile(outputPath, data, 0644)
}

// SanitizeFileName removes invalid characters from filenames
func SanitizeFileName(name string) string {
	if name == "" {
		return "Unknown"
	}

	// Remove characters invalid on Windows/Linux/macOS
	invalid := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	sanitized := invalid.ReplaceAllString(name, "")

	// Replace multiple spaces with single space
	spaces := regexp.MustCompile(`\s+`)
	sanitized = spaces.ReplaceAllString(sanitized, " ")

	// Remove leading/trailing dots and spaces
	sanitized = strings.Trim(sanitized, ". ")

	// Limit length
	if len(sanitized) > 200 {
		sanitized = sanitized[:200]
	}

	if sanitized == "" {
		return "Unknown"
	}

	return sanitized
}

// formatFileName generates filename based on format template
func (t *TidalHifiService) formatFileName(track *TidalHifiTrackResponse, artistName string) string {
	format := t.options.FileNameFormat
	if format == "" {
		format = "{artist} - {title}"
	}

	// Replace placeholders
	result := format
	result = strings.ReplaceAll(result, "{artist}", artistName)
	result = strings.ReplaceAll(result, "{title}", track.Title)
	result = strings.ReplaceAll(result, "{album}", track.Album.Title)
	result = strings.ReplaceAll(result, "{track}", fmt.Sprintf("%02d", track.TrackNumber))
	result = strings.ReplaceAll(result, "{isrc}", track.ISRC)

	return SanitizeFileName(result)
}

// FormatCoverUUID converts a Tidal cover UUID to URL path format
func FormatCoverUUID(uuid string) string {
	// Tidal uses UUIDs like "abc-def-ghi" that need to be "abc/def/ghi"
	return strings.ReplaceAll(uuid, "-", "/")
}

// DownloadedFileInfo represents metadata for a downloaded file
type DownloadedFileInfo struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
	Title   string `json:"title"`
	Artist  string `json:"artist"`
	Album   string `json:"album"`
}

// ListFLACFiles lists all FLAC files in the given directory
func ListFLACFiles(folder string) ([]DownloadedFileInfo, error) {
	var files []DownloadedFileInfo

	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".flac") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		filePath := filepath.Join(folder, name)

		// Try to parse metadata from filename (format: "Artist - Title.flac")
		title, artist, album := "", "", ""
		baseName := strings.TrimSuffix(name, ".flac")
		if parts := strings.SplitN(baseName, " - ", 2); len(parts) == 2 {
			artist = parts[0]
			title = parts[1]
		} else {
			title = baseName
		}

		files = append(files, DownloadedFileInfo{
			Path:    filePath,
			Name:    name,
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02T15:04:05Z07:00"),
			Title:   title,
			Artist:  artist,
			Album:   album,
		})
	}

	return files, nil
}

// DeleteFile deletes a file from the filesystem
func DeleteFile(path string) error {
	// Security check: only allow deleting FLAC files
	if !strings.HasSuffix(strings.ToLower(path), ".flac") {
		return fmt.Errorf("can only delete FLAC files")
	}

	return os.Remove(path)
}
