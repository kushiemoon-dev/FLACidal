package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"strconv"
	"strings"
	"time"

	core "github.com/kushiemoon-dev/flacidal-core"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// =============================================================================
// Source Health & Soulseek Methods (exposed to frontend)
// =============================================================================

// GetSourceHealth runs an on-demand capability probe for each registered source.
// Returns per-source status: online, degraded, dead, or untested.
// Called from the Settings Status tab on user request only — never polled.
// GetSourceHealth returns pool state for all sources without making any network
// requests. States reflect real failures observed during actual downloads, not
// synthetic probes — avoids the WebKitGTK signal-handler conflict on Linux.
func (a *App) GetSourceHealth() []core.SourceHealth {
	var results []core.SourceHealth

	if a.downloader != nil {
		snaps := a.downloader.PoolSnapshot()
		results = append(results, core.SourceHealth{
			Name:        "tidal",
			DisplayName: "Tidal HiFi",
			Status:      poolSnapshotStatus(snaps),
			Endpoints:   snaps,
		})
	}

	if a.qobuzSource != nil {
		snaps := a.qobuzSource.ProxyPoolSnapshot()
		results = append(results, core.SourceHealth{
			Name:        "qobuz",
			DisplayName: "Qobuz",
			Status:      poolSnapshotStatus(snaps),
			Endpoints:   snaps,
		})
	}

	if a.amazonSource != nil {
		snaps := a.amazonSource.PoolSnapshot()
		results = append(results, core.SourceHealth{
			Name:        "amazon",
			DisplayName: "Amazon Music",
			Status:      poolSnapshotStatus(snaps),
			Endpoints:   snaps,
		})
	}

	if a.soulseekSource != nil {
		status := "dead"
		reason := ""
		if a.soulseekSource.IsAvailable() {
			status = "online"
		} else if a.config != nil && (a.config.SoulseekUsername == "" || a.config.SoulseekPassword == "") {
			reason = "credentials not configured"
		} else {
			reason = "sldl not installed"
		}
		results = append(results, core.SourceHealth{
			Name:        "soulseek",
			DisplayName: "Soulseek",
			Status:      status,
			Reason:      reason,
		})
	}

	return results
}

// poolSnapshotStatus maps an endpoint pool snapshot to a SourceHealth status string.
func poolSnapshotStatus(snaps []core.EndpointStat) string {
	if len(snaps) == 0 {
		return "untested"
	}
	live := 0
	for _, ep := range snaps {
		if ep.State == "live" || ep.State == "probation" {
			live++
		}
	}
	switch {
	case live == 0:
		return "dead"
	case live < len(snaps):
		return "degraded"
	default:
		return "online"
	}
}

// InstallSldl downloads and installs the pinned sldl (sockseek) binary, emitting progress events.
// After install, re-initializes the Soulseek source so IsAvailable() flips without a restart.
func (a *App) InstallSldl() error {
	progressCh := make(chan core.SldlInstallProgress, 10)

	go func() {
		for p := range progressCh {
			runtime.EventsEmit(a.ctx, "sldl-install-progress", p)
		}
	}()

	if err := core.InstallSldl(progressCh); err != nil {
		if a.logBuffer != nil {
			a.logBuffer.Error("sldl install failed: " + err.Error())
		}
		return err
	}

	// Remove quarantine attribute on macOS and ensure executable bit
	sldlPath := core.GetSldlPath()
	if err := ensureSldlExecutable(sldlPath); err != nil {
		a.logBuffer.Warn(fmt.Sprintf("sldl binary may not be executable: %v", err))
	}

	// Re-initialize Soulseek source so IsAvailable() flips without restart
	if a.config != nil {
		username := a.config.SoulseekUsername
		password := a.config.SoulseekPassword
		a.soulseekSource = core.NewSoulseekSource(sldlPath, username, password)
		a.soulseekSource.SetLogger(a.logBuffer)
		if a.config.SoulseekEnabled && a.soulseekSource.IsAvailable() {
			a.sourceManager.RegisterSource(a.soulseekSource)
			a.logBuffer.Info("Soulseek source registered after sldl install")
		}
	}

	if a.logBuffer != nil {
		a.logBuffer.Info("sldl installed successfully to " + sldlPath)
	}
	return nil
}

// GetSldlStatus checks if the sldl binary is installed and returns its version
func (a *App) GetSldlStatus() map[string]interface{} {
	sldlPath := ""
	if a.config != nil {
		sldlPath = a.config.SoulseekBinaryPath
	}
	return SldlStatus(sldlPath)
}

// SldlStatus checks if the sldl binary is installed and returns its version.
// Shared by the desktop (Wails) and HTTP server APIs (same sharing pattern as
// ConvertTidalSearchResults / SearchDeezerTracks in app_search.go). binaryPath
// may be empty, in which case the platform default path is used.
func SldlStatus(binaryPath string) map[string]interface{} {
	sldlPath := binaryPath
	if sldlPath == "" {
		sldlPath = defaultSldlPath()
	}

	if _, err := os.Stat(sldlPath); os.IsNotExist(err) {
		return map[string]interface{}{
			"installed": false,
			"path":      sldlPath,
			"version":   "",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, sldlPath, "--version").Output()
	version := ""
	if err == nil {
		version = strings.TrimSpace(string(out))
	}

	return map[string]interface{}{
		"installed": true,
		"path":      sldlPath,
		"version":   version,
	}
}

// TestSoulseekConnection verifies Soulseek credentials by running a quick search via sldl.
// Success is determined by detecting an explicit "Logged in" message in verbose output,
// which is emitted before any search results and is independent of firewall/inbound connectivity.
func (a *App) TestSoulseekConnection(username, password string) map[string]interface{} {
	sldlPath := ""
	if a.config != nil {
		sldlPath = a.config.SoulseekBinaryPath
	}
	logf := func(level, msg string) {
		if a.logBuffer == nil {
			return
		}
		if level == "warn" {
			a.logBuffer.Warn(msg)
		} else {
			a.logBuffer.Info(msg)
		}
	}
	return TestSoulseekLogin(sldlPath, username, password, logf)
}

// TestSoulseekLogin is the shared implementation of TestSoulseekConnection,
// used by both the desktop (Wails) and HTTP server APIs (same sharing
// pattern as ConvertTidalSearchResults / SearchDeezerTracks in
// app_search.go). binaryPath may be empty, in which case the platform
// default path is used. logf receives best-effort diagnostic lines ("info"
// or "warn" level) — pass nil to skip logging (the password is never
// logged either way).
func TestSoulseekLogin(binaryPath, username, password string, logf func(level, msg string)) map[string]interface{} {
	sldlPath := binaryPath
	if sldlPath == "" {
		sldlPath = defaultSldlPath()
	}
	log := func(level, msg string) {
		if logf != nil {
			logf(level, msg)
		}
	}

	if _, err := os.Stat(sldlPath); os.IsNotExist(err) {
		return map[string]interface{}{"success": false, "message": "sldl not found"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// -v (verbose) makes sldl print "Logged in <user>" on a dedicated line immediately
	// after authentication succeeds — independently of whether search results arrive via
	// inbound P2P connections. Without -v the only success signal was result lines ([...]),
	// which require inbound connectivity and are blocked by the default Windows/macOS firewall.
	if err := ensureSldlExecutable(sldlPath); err != nil {
		log("warn", fmt.Sprintf("sldl binary may not be executable: %v", err))
	}
	cmd := exec.CommandContext(ctx, sldlPath,
		"test",
		"--user", username,
		"--pass", password,
		"--listen-port", "49996",
		"--print", "results",
		"--no-progress",
		"-v",
	)
	out, execErr := cmd.CombinedOutput()
	rawOutput := strings.ToLower(string(out))

	// Surface diagnostics in the in-app terminal (password is never logged)
	log("info", fmt.Sprintf("Soulseek: testing connection for user %q", username))
	if execErr != nil {
		log("warn", fmt.Sprintf("Soulseek: sldl process error: %v", execErr))
	}
	if trimmed := strings.TrimSpace(string(out)); trimmed != "" {
		log("info", "Soulseek: sldl output:\n"+trimmed)
	}

	if ctx.Err() == context.DeadlineExceeded {
		return map[string]interface{}{"success": false, "message": "Connection timed out"}
	}

	// sldl produced no output — process failed to start (Gatekeeper/AV/permissions)
	if strings.TrimSpace(rawOutput) == "" && execErr != nil {
		hint := "ensure sldl is not blocked by antivirus or SmartScreen"
		if goruntime.GOOS == "darwin" {
			hint = "run: xattr -d com.apple.quarantine " + sldlPath
		}
		return map[string]interface{}{"success": false, "message": fmt.Sprintf("sldl failed to start — %s", hint)}
	}
	if strings.TrimSpace(rawOutput) == "" {
		return map[string]interface{}{"success": false, "message": "sldl produced no output — verify the binary is valid"}
	}

	// .NET runtime not installed (framework-dependent build downloaded instead of self-contained)
	if strings.Contains(rawOutput, "must install") && strings.Contains(rawOutput, ".net") {
		return map[string]interface{}{"success": false, "message": ".NET runtime missing — download the self-contained sldl build from github.com/fiso64/slsk-batchdl/releases"}
	}

	// Auth failures — check before the success path so a rejected login is never
	// misreported as a network error.
	authErrors := []string{
		"wrong password", "invalid password", "incorrect password",
		"login failed", "failed to log in", "cannot login", "could not log in",
		"login rejected", "authentication failed", "invalidpass",
	}
	for _, kw := range authErrors {
		if strings.Contains(rawOutput, kw) {
			return map[string]interface{}{"success": false, "message": "Invalid credentials"}
		}
	}

	// Explicit login success emitted by sldl -v: "Logged in <username>"
	if strings.Contains(rawOutput, "logged in ") {
		return map[string]interface{}{"success": true, "message": "Logged in"}
	}

	// Network / connectivity failures
	networkErrors := []string{
		"could not connect", "connection refused", "unable to connect",
		"no such host", "network is unreachable", "name resolution",
		"connect: connection", "dial tcp",
	}
	for _, kw := range networkErrors {
		if strings.Contains(rawOutput, kw) {
			return map[string]interface{}{"success": false, "message": "Connection failed — check network"}
		}
	}

	return map[string]interface{}{"success": false, "message": "Connection failed — check network or credentials"}
}

// =============================================================================
// Source Manager Methods (exposed to frontend)
// =============================================================================

// GetAvailableSources returns info about all registered music sources
func (a *App) GetAvailableSources() []core.SourceInfo {
	return a.sourceManager.GetSourcesInfo()
}

// GetPreferredSource returns the currently preferred source name
func (a *App) GetPreferredSource() string {
	source, ok := a.sourceManager.GetPreferredSource()
	if ok {
		return source.Name()
	}
	return "tidal"
}

// SetPreferredSource sets the preferred source
func (a *App) SetPreferredSource(sourceName string) {
	a.sourceManager.SetPreferredSource(sourceName)
	if a.logBuffer != nil {
		a.logBuffer.Info(fmt.Sprintf("Preferred source set to: %s", sourceName))
	}
}

// DetectSourceFromURL identifies which source can handle a URL
func (a *App) DetectSourceFromURL(rawURL string) map[string]interface{} {
	result := map[string]interface{}{
		"detected":    false,
		"source":      "",
		"displayName": "",
		"contentType": "",
		"id":          "",
		"available":   false,
	}

	source, err := a.sourceManager.DetectSource(rawURL)
	if err != nil {
		return result
	}

	id, contentType, err := source.ParseURL(rawURL)
	if err != nil {
		return result
	}

	result["detected"] = true
	result["source"] = source.Name()
	result["displayName"] = source.DisplayName()
	result["contentType"] = contentType
	result["id"] = id
	result["available"] = source.IsAvailable()

	return result
}

// FetchContentFromURL fetches content info from any supported source URL
// PickOdesliCandidate returns the first of links' URLs that a source
// registered on sm can parse, preferring Tidal then Deezer. Amazon is
// deliberately excluded: AmazonSource.ParseURL always errors (it's
// download/ISRC-search only, never URL-routable), so it could never be
// picked here anyway. Exported so internal/api's own SourceManager-backed
// server can share this selection logic instead of duplicating it.
func PickOdesliCandidate(sm *core.SourceManager, links *core.OdesliLinks) (string, bool) {
	for _, candidate := range []string{links.Tidal, links.Deezer} {
		if candidate == "" {
			continue
		}
		if _, err := sm.DetectSource(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}

// ResolveViaOdesli looks up rawURL on Odesli/song.link for input FLACidal has
// no native parser for (Spotify already has one — this covers Apple Music,
// YouTube Music, Deezer short links, etc.) and returns the first resolved
// link a source registered on sm can actually parse. Skips the Odesli call
// entirely when no source is registered, since nothing could consume the
// result anyway.
func ResolveViaOdesli(sm *core.SourceManager, rawURL string) (string, error) {
	if len(sm.GetSourcesInfo()) == 0 {
		return "", fmt.Errorf("no source found for URL: %s", rawURL)
	}
	links, err := core.ResolveOdesliLinks(rawURL)
	if err != nil {
		return "", err
	}
	if candidate, ok := PickOdesliCandidate(sm, links); ok {
		return candidate, nil
	}
	return "", fmt.Errorf("odesli resolved %s but no supported source could parse the result", rawURL)
}

func (a *App) FetchContentFromURL(rawURL string) (map[string]interface{}, error) {
	resolvedViaOdesli := false
	source, err := a.sourceManager.DetectSource(rawURL)
	if err != nil {
		resolvedURL, rerr := ResolveViaOdesli(a.sourceManager, rawURL)
		if rerr != nil {
			return nil, rerr
		}
		if a.logBuffer != nil {
			a.logBuffer.Info(fmt.Sprintf("Resolved %s via Odesli to %s", rawURL, resolvedURL))
		}
		rawURL = resolvedURL
		resolvedViaOdesli = true
		source, err = a.sourceManager.DetectSource(rawURL)
		if err != nil {
			return nil, err
		}
	}

	id, contentType, err := source.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"source": source.Name(),
		"type":   contentType,
		"id":     id,
	}
	if resolvedViaOdesli {
		result["resolvedVia"] = "odesli"
	}

	// Helper to convert SourceTrack to frontend-compatible format
	convertTrack := func(t core.SourceTrack) map[string]interface{} {
		// Convert ID to int if possible, otherwise use string
		trackID, _ := strconv.Atoi(t.ID)
		artists := t.Artist
		if len(t.Artists) > 0 {
			artists = strings.Join(t.Artists, ", ")
		}
		return map[string]interface{}{
			"id":          trackID,
			"title":       t.Title,
			"artist":      t.Artist,
			"artists":     artists,
			"album":       t.Album,
			"duration":    t.Duration,
			"trackNumber": t.TrackNumber,
			"coverUrl":    t.CoverURL,
			"explicit":    t.Explicit,
			"isrc":        t.ISRC,
		}
	}

	convertTracks := func(tracks []core.SourceTrack) []map[string]interface{} {
		result := make([]map[string]interface{}, len(tracks))
		for i, t := range tracks {
			result[i] = convertTrack(t)
		}
		return result
	}

	switch contentType {
	case "track":
		track, err := source.GetTrack(id)
		if err != nil {
			return nil, err
		}
		result["title"] = track.Title
		result["creator"] = track.Artist
		result["coverUrl"] = track.CoverURL
		result["tracks"] = convertTracks([]core.SourceTrack{*track})

	case "album":
		album, err := source.GetAlbum(id)
		if err != nil {
			return nil, err
		}
		result["title"] = album.Title
		result["creator"] = album.Artist
		result["coverUrl"] = album.CoverURL
		result["tracks"] = convertTracks(album.Tracks)

	case "playlist":
		playlist, err := source.GetPlaylist(id)
		if err != nil {
			return nil, err
		}
		result["title"] = playlist.Title
		result["creator"] = playlist.Creator
		result["coverUrl"] = playlist.CoverURL
		result["tracks"] = convertTracks(playlist.Tracks)

	case "mix":
		mix, err := a.downloader.GetMixFromProxy(id)
		if err != nil {
			return nil, err
		}
		result["title"] = mix.Title
		result["creator"] = mix.Creator
		result["coverUrl"] = mix.CoverURL
		tidalTracks := make([]core.SourceTrack, len(mix.Tracks))
		for i, t := range mix.Tracks {
			tidalTracks[i] = core.SourceTrack{
				ID:          strconv.Itoa(t.ID),
				Title:       t.Title,
				Artist:      t.Artist,
				Artists:     []string{t.Artists},
				Album:       t.Album,
				ISRC:        t.ISRC,
				Duration:    t.Duration,
				TrackNumber: t.TrackNum,
				CoverURL:    t.CoverURL,
				Explicit:    t.Explicit,
				SourceURL:   t.TidalURL,
				Source:      "tidal",
			}
		}
		result["tracks"] = convertTracks(tidalTracks)
	}

	if a.logBuffer != nil {
		a.logBuffer.Info(fmt.Sprintf("Fetched %s from %s: %s", contentType, source.DisplayName(), id))
	}

	return result, nil
}

// ExpandDiscographyURL detects a Spotify discography URL and returns all album URLs for that artist.
// Returns an error if the URL is not a valid Spotify discography URL.
func (a *App) ExpandDiscographyURL(rawURL string) ([]string, error) {
	info := core.ParseDiscographyURL(rawURL)
	if info == nil {
		return nil, fmt.Errorf("not a Spotify discography URL: %s", rawURL)
	}
	if a.spotifySearch == nil {
		return nil, fmt.Errorf("Spotify client not initialized")
	}
	urls, err := a.spotifySearch.FetchDiscographyAlbumURLs(info.ArtistID, info.Kind)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch discography for artist %s: %w", info.ArtistID, err)
	}
	if a.logBuffer != nil {
		a.logBuffer.Info(fmt.Sprintf("Discography expanded: %d albums for artist %s (kind=%s)", len(urls), info.ArtistID, info.Kind))
	}
	return urls, nil
}

// QueueDiscographyAlbums resolves a list of Spotify album URLs to Tidal albums and queues them.
// For each URL it fetches Spotify metadata (title + artist), searches Tidal, then queues the
// best-matching album via the TidalHifi proxy. Returns the count of successfully queued albums.
func (a *App) QueueDiscographyAlbums(spotifyAlbumURLs []string, outputDir string) (int, error) {
	if a.downloadManager == nil {
		return 0, fmt.Errorf("download manager not initialized")
	}
	if outputDir == "" {
		return 0, fmt.Errorf("no output directory specified")
	}
	if a.spotifySearch == nil {
		return 0, fmt.Errorf("Spotify client not initialized")
	}
	if a.tidalSource == nil {
		return 0, fmt.Errorf("Tidal source not initialized")
	}

	tidalClient := a.tidalSource.GetAPIClient()
	spotifyIDRe := regexp.MustCompile(`open\.spotify\.com/album/([^/?#]+)`)

	queued := 0
	for _, albumURL := range spotifyAlbumURLs {
		m := spotifyIDRe.FindStringSubmatch(albumURL)
		if m == nil {
			continue
		}
		spotifyAlbumID := m[1]

		albumName, artistName, err := a.spotifySearch.GetAlbumMetadata(spotifyAlbumID)
		if err != nil {
			if a.logBuffer != nil {
				a.logBuffer.Warn(fmt.Sprintf("Discography: skipping %s — Spotify metadata failed: %v", spotifyAlbumID, err))
			}
			continue
		}

		query := albumName + " " + artistName
		tidalAlbums, err := tidalClient.SearchAlbums(query, 5)
		if err != nil || len(tidalAlbums) == 0 {
			if a.logBuffer != nil {
				a.logBuffer.Warn(fmt.Sprintf("Discography: no Tidal match for %q by %s", albumName, artistName))
			}
			continue
		}

		tidalAlbum := tidalAlbums[0]
		albumIDStr := strconv.Itoa(tidalAlbum.ID)

		album, err := a.downloader.GetAlbumFromProxy(albumIDStr)
		if err != nil {
			if a.logBuffer != nil {
				a.logBuffer.Warn(fmt.Sprintf("Discography: could not fetch Tidal album %s: %v", albumIDStr, err))
			}
			continue
		}

		artistFolder := core.SanitizeFileName(tidalAlbum.Artist)
		if artistFolder == "" {
			artistFolder = core.SanitizeFileName(artistName)
		}
		albumDir := filepath.Join(outputDir, artistFolder, core.SanitizeFileName(tidalAlbum.Title))
		if err := os.MkdirAll(albumDir, 0755); err != nil {
			continue
		}

		n := a.downloadManager.QueueMultiple(album.Tracks, albumDir)
		queued += n
	}

	return queued, nil
}

// GetSourceTrack fetches a track from a specific source
func (a *App) GetSourceTrack(sourceName, trackID string) (*core.SourceTrack, error) {
	source, ok := a.sourceManager.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("source not found: %s", sourceName)
	}
	return source.GetTrack(trackID)
}

// GetSourceAlbum fetches an album from a specific source
func (a *App) GetSourceAlbum(sourceName, albumID string) (*core.SourceAlbum, error) {
	source, ok := a.sourceManager.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("source not found: %s", sourceName)
	}
	return source.GetAlbum(albumID)
}

// GetSourcePlaylist fetches a playlist from a specific source
func (a *App) GetSourcePlaylist(sourceName, playlistID string) (*core.SourcePlaylist, error) {
	source, ok := a.sourceManager.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("source not found: %s", sourceName)
	}
	return source.GetPlaylist(playlistID)
}

// UpdateQobuzCredentials updates Qobuz credentials
func (a *App) UpdateQobuzCredentials(appID, appSecret, authToken string) error {
	if a.qobuzSource == nil {
		a.qobuzSource = core.NewQobuzSource(appID, appSecret)
	}
	a.qobuzSource.SetCredentials(appID, appSecret, authToken)

	if a.config == nil {
		a.config = &core.Config{}
	}
	// Update config
	a.config.QobuzAppID = appID
	a.config.QobuzAppSecret = appSecret
	a.config.QobuzAuthToken = authToken
	a.config.QobuzEnabled = appID != "" && appSecret != ""

	if err := core.SaveConfig(a.config); err != nil {
		return err
	}

	if a.logBuffer != nil {
		if a.config.QobuzEnabled {
			a.logBuffer.Success("Qobuz credentials updated")
		} else {
			a.logBuffer.Info("Qobuz disabled")
		}
	}

	return nil
}

// IsQobuzConfigured checks if Qobuz is properly configured
func (a *App) IsQobuzConfigured() bool {
	return a.qobuzSource != nil && a.qobuzSource.IsAvailable()
}
