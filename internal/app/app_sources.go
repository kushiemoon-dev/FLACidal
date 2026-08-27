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

// GetSourceHealth reports one of: online, degraded, dead, or untested, per
// source. It only runs when the user opens the Settings Status tab (never
// polled) and reads pool state without issuing network requests — reported
// states come from real failures seen during actual downloads rather than
// synthetic probes, which sidesteps the WebKitGTK signal-handler conflict on
// Linux.
func (a *App) GetSourceHealth() []core.SourceHealth {
	var results []core.SourceHealth

	if a.downloader != nil {
		snaps := a.downloader.PoolSnapshot()
		tier1Snaps := a.downloader.Tier1PoolSnapshot()
		tier1, failureKind, retryETASecs := poolSnapshotTier1Status(snaps, tier1Snaps, a.downloader.NextRevivalETASecs)
		results = append(results, core.SourceHealth{
			Name:         "tidal",
			DisplayName:  "Tidal HiFi",
			Status:       poolSnapshotStatus(snaps),
			Endpoints:    snaps,
			Tier1:        tier1,
			FailureKind:  failureKind,
			RetryETASecs: retryETASecs,
		})
	}

	if a.qobuzSource != nil {
		snaps := a.qobuzSource.ProxyPoolSnapshot()
		tier1Snaps := a.qobuzSource.Tier1ProxyPoolSnapshot()
		tier1, failureKind, retryETASecs := poolSnapshotTier1Status(snaps, tier1Snaps, a.qobuzSource.ProxyNextRevivalETASecs)
		results = append(results, core.SourceHealth{
			Name:         "qobuz",
			DisplayName:  "Qobuz",
			Status:       poolSnapshotStatus(snaps),
			Endpoints:    snaps,
			Tier1:        tier1,
			FailureKind:  failureKind,
			RetryETASecs: retryETASecs,
		})
	}

	if a.amazonSource != nil {
		snaps := a.amazonSource.PoolSnapshot()
		tier1Snaps := a.amazonSource.Tier1PoolSnapshot()
		tier1, failureKind, retryETASecs := poolSnapshotTier1Status(snaps, tier1Snaps, a.amazonSource.NextRevivalETASecs)
		results = append(results, core.SourceHealth{
			Name:         "amazon",
			DisplayName:  "Amazon Music",
			Status:       poolSnapshotStatus(snaps),
			Endpoints:    snaps,
			Tier1:        tier1,
			FailureKind:  failureKind,
			RetryETASecs: retryETASecs,
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

// poolSnapshotTier1Status derives a source's Tier1 status, FailureKind, and
// RetryETASecs purely from already-fetched pool snapshots — no network
// calls, matching this file's no-live-probe constraint (see GetSourceHealth's
// doc comment). tier1Snaps empty means Tier1 stays nil (not configured).
// FailureKindUpstream is only reported once no tier1 entry is healthy AND
// every entry in the full snapshot (both tiers) is dead — the same
// no-live-endpoint condition Core's own live probes (probeQobuz, probeAmazon,
// ProbeTidalService) use, just computed from a snapshot instead of a fresh
// check. nextRevivalETASecs is only invoked in that case, since RetryETASecs
// is meaningless otherwise.
func poolSnapshotTier1Status(snaps, tier1Snaps []core.EndpointStat, nextRevivalETASecs func() int) (*core.Tier1Status, core.FailureKind, int) {
	var tier1 *core.Tier1Status
	tier1Healthy := false
	if len(tier1Snaps) > 0 {
		for _, ep := range tier1Snaps {
			if ep.State != "dead" {
				tier1Healthy = true
				break
			}
		}
		tier1 = &core.Tier1Status{
			Configured: true,
			Healthy:    tier1Healthy,
			Endpoints:  tier1Snaps,
		}
	}

	allDead := true
	for _, ep := range snaps {
		if ep.State != "dead" {
			allDead = false
			break
		}
	}

	if !tier1Healthy && allDead {
		return tier1, core.FailureKindUpstream, nextRevivalETASecs()
	}
	return tier1, core.FailureKindNone, 0
}

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

	// Strip the quarantine attribute on macOS and make sure the executable bit is set
	sldlPath := core.GetSldlPath()
	if err := EnsureSldlExecutable(sldlPath); err != nil {
		a.logBuffer.Warn(fmt.Sprintf("sldl binary might not be runnable: %v", err))
	}

	// Rebuild the Soulseek source so IsAvailable() picks up the change without a restart
	if a.config != nil {
		username := a.config.SoulseekUsername
		password := a.config.SoulseekPassword
		a.soulseekSource = core.NewSoulseekSource(sldlPath, username, password)
		a.soulseekSource.SetLogger(a.logBuffer)
		if a.config.SoulseekEnabled && a.soulseekSource.IsAvailable() {
			a.sourceManager.RegisterSource(a.soulseekSource)
			a.logBuffer.Info("Soulseek source registered following the sldl install")
		}
	}

	if a.logBuffer != nil {
		a.logBuffer.Info("sldl installed to " + sldlPath)
	}
	return nil
}

func (a *App) GetSldlStatus() map[string]interface{} {
	sldlPath := ""
	if a.config != nil {
		sldlPath = a.config.SoulseekBinaryPath
	}
	return SldlStatus(sldlPath)
}

// SldlStatus is shared by the desktop (Wails) and HTTP server APIs, the same
// sharing pattern ConvertTidalSearchResults / SearchDeezerTracks use in
// app_search.go. binaryPath can be left empty, in which case the platform
// default path is used.
func SldlStatus(binaryPath string) map[string]interface{} {
	sldlPath := binaryPath
	if sldlPath == "" {
		sldlPath = DefaultSldlPath()
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

// TestSoulseekConnection judges success by spotting an explicit "Logged in"
// message in the verbose sldl output, which appears before any search
// results and doesn't depend on firewall/inbound connectivity.
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

// TestSoulseekLogin backs TestSoulseekConnection and is shared by both the
// desktop (Wails) and HTTP server APIs — the same sharing pattern
// ConvertTidalSearchResults / SearchDeezerTracks use in app_search.go.
// binaryPath can be left empty, in which case the platform default path is
// used. logf gets best-effort diagnostic lines at "info" or "warn" level;
// pass nil to skip logging entirely (the password is never logged either way).
func TestSoulseekLogin(binaryPath, username, password string, logf func(level, msg string)) map[string]interface{} {
	sldlPath := binaryPath
	if sldlPath == "" {
		sldlPath = DefaultSldlPath()
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

	// -v (verbose) has sldl print "Logged in <user>" on its own line right after
	// authentication succeeds, regardless of whether search results come back over
	// inbound P2P connections. Without -v the only success signal was the result
	// lines ([...]), which need inbound connectivity that's blocked by the default
	// Windows/macOS firewall.
	if err := EnsureSldlExecutable(sldlPath); err != nil {
		log("warn", fmt.Sprintf("sldl binary might not be runnable: %v", err))
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

	// The password itself is never logged, even in these diagnostics.
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

	// No output at all means the process never really started (Gatekeeper/AV/permissions)
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

	// The .NET runtime isn't installed (a framework-dependent build got downloaded instead of a self-contained one)
	if strings.Contains(rawOutput, "must install") && strings.Contains(rawOutput, ".net") {
		return map[string]interface{}{"success": false, "message": ".NET runtime missing — download the self-contained sldl build from github.com/fiso64/slsk-batchdl/releases"}
	}

	// Check auth failures before the success path, so a rejected login never
	// gets misreported as a network error.
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

	// sldl -v emits an explicit "Logged in <username>" line on success
	if strings.Contains(rawOutput, "logged in ") {
		return map[string]interface{}{"success": true, "message": "Logged in"}
	}

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

func (a *App) GetAvailableSources() []core.SourceInfo {
	return a.sourceManager.GetSourcesInfo()
}

func (a *App) GetPreferredSource() string {
	source, ok := a.sourceManager.GetPreferredSource()
	if ok {
		return source.Name()
	}
	return "tidal"
}

func (a *App) SetPreferredSource(sourceName string) {
	a.sourceManager.SetPreferredSource(sourceName)
	if a.logBuffer != nil {
		a.logBuffer.Info(fmt.Sprintf("Preferred source set to: %s", sourceName))
	}
}

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

// PickOdesliCandidate picks the first URL among links that a source
// registered on sm is able to parse, favoring Tidal ahead of Deezer. Amazon
// is deliberately left out: AmazonSource.ParseURL always errors (it only
// supports download/ISRC search, never URL routing), so it could never be
// chosen here regardless. This is exported so internal/api's own
// SourceManager-backed server can reuse this selection logic rather than
// duplicating it.
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

// ResolveViaOdesli looks rawURL up on Odesli/song.link for inputs FLACidal
// has no native parser for — Spotify already has one, so this covers Apple
// Music, YouTube Music, Deezer short links, and similar — and returns the
// first resolved link that a source registered on sm can actually parse. The
// Odesli call is skipped entirely when no source is registered, since
// nothing would be able to consume the result anyway.
func ResolveViaOdesli(sm *core.SourceManager, rawURL string) (string, error) {
	if len(sm.GetSourcesInfo()) == 0 {
		return "", fmt.Errorf("no source registered that can handle URL: %s", rawURL)
	}
	links, err := core.ResolveOdesliLinks(rawURL)
	if err != nil {
		return "", err
	}
	if candidate, ok := PickOdesliCandidate(sm, links); ok {
		return candidate, nil
	}
	return "", fmt.Errorf("odesli resolved %s, but none of the registered sources could parse the result", rawURL)
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

	convertTrack := func(t core.SourceTrack) map[string]interface{} {
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
				a.logBuffer.Warn(fmt.Sprintf("Discography: skipping %s, Spotify metadata lookup failed: %v", spotifyAlbumID, err))
			}
			continue
		}

		query := albumName + " " + artistName
		tidalAlbums, err := tidalClient.SearchAlbums(query, 5)
		if err != nil || len(tidalAlbums) == 0 {
			if a.logBuffer != nil {
				a.logBuffer.Warn(fmt.Sprintf("Discography: found no Tidal match for %q by %s", albumName, artistName))
			}
			continue
		}

		tidalAlbum := tidalAlbums[0]
		albumIDStr := strconv.Itoa(tidalAlbum.ID)

		album, err := a.downloader.GetAlbumFromProxy(albumIDStr)
		if err != nil {
			if a.logBuffer != nil {
				a.logBuffer.Warn(fmt.Sprintf("Discography: unable to fetch Tidal album %s: %v", albumIDStr, err))
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

func (a *App) GetSourceTrack(sourceName, trackID string) (*core.SourceTrack, error) {
	source, ok := a.sourceManager.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("source not found: %s", sourceName)
	}
	return source.GetTrack(trackID)
}

func (a *App) GetSourceAlbum(sourceName, albumID string) (*core.SourceAlbum, error) {
	source, ok := a.sourceManager.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("source not found: %s", sourceName)
	}
	return source.GetAlbum(albumID)
}

func (a *App) GetSourcePlaylist(sourceName, playlistID string) (*core.SourcePlaylist, error) {
	source, ok := a.sourceManager.GetSource(sourceName)
	if !ok {
		return nil, fmt.Errorf("source not found: %s", sourceName)
	}
	return source.GetPlaylist(playlistID)
}

func (a *App) UpdateQobuzCredentials(appID, appSecret, authToken string) error {
	if a.qobuzSource == nil {
		a.qobuzSource = core.NewQobuzSource(appID, appSecret)
	}
	a.qobuzSource.SetCredentials(appID, appSecret, authToken)

	if a.config == nil {
		a.config = &core.Config{}
	}
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

func (a *App) IsQobuzConfigured() bool {
	return a.qobuzSource != nil && a.qobuzSource.IsAvailable()
}
