package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// HealthSources are the sources whose health is reported. Each field may be
// nil. Shared by the Wails App and the headless server so both build the same
// statuses.
type HealthSources struct {
	Downloader *core.TidalHifiService
	Qobuz      *core.QobuzSource
	Amazon     *core.AmazonSource
	Soulseek   *core.SoulseekNativeSource
	Config     *core.Config
}

func (a *App) healthSources() HealthSources {
	cfg := a.cfg()
	var soulseek *core.SoulseekNativeSource
	if a.core != nil {
		soulseek = a.core.SoulseekSource()
	}
	return HealthSources{
		Downloader: a.downloader,
		Qobuz:      a.qobuzSource,
		Amazon:     a.amazonSource,
		Soulseek:   soulseek,
		Config:     &cfg,
	}
}

// GetSourceHealth reports one of: online, degraded, dead, or untested, per
// source. It reads pool state in memory and issues no network request;
// ProbeEndpoints is the action that does (the "Test endpoints" button).
func (a *App) GetSourceHealth() []core.SourceHealth {
	return BuildSourceHealth(a.healthSources())
}

// ProbeEndpoints sends one reference request to every endpoint of every pool,
// then returns the refreshed source health. A failed probe only records its
// result, it never changes routing. It runs on a plain Go goroutine (Wails
// runs each bound method on its own) and uses no Wails runtime.
func (a *App) ProbeEndpoints() []core.SourceHealth {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	h := a.healthSources()
	ProbeAllEndpoints(ctx, h, 0)
	return BuildSourceHealth(h)
}

// ProbeAllEndpoints probes the pools of every non-nil source in parallel.
// skipRecent > 0 skips endpoints used by a real request within that window.
// Soulseek is never probed automatically: a login probe risks a ban.
func ProbeAllEndpoints(ctx context.Context, h HealthSources, skipRecent time.Duration) {
	var wg sync.WaitGroup
	run := func(f func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f()
		}()
	}
	if h.Downloader != nil {
		run(func() { h.Downloader.ProbeEndpoints(ctx, skipRecent) })
	}
	if h.Qobuz != nil {
		run(func() { h.Qobuz.ProbeProxyEndpoints(ctx, skipRecent) })
	}
	if h.Amazon != nil {
		run(func() { h.Amazon.ProbeEndpoints(ctx, skipRecent) })
	}
	wg.Wait()
}

// BuildSourceHealth builds the health of every non-nil source from in-memory
// pool state.
func BuildSourceHealth(h HealthSources) []core.SourceHealth {
	var results []core.SourceHealth

	if h.Downloader != nil {
		snaps := h.Downloader.PoolSnapshot()
		tier1Snaps := h.Downloader.Tier1PoolSnapshot()
		tier1, failureKind, retryETASecs := poolSnapshotTier1Status(snaps, tier1Snaps, h.Downloader.NextRevivalETASecs)
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

	if h.Qobuz != nil {
		snaps := h.Qobuz.ProxyPoolSnapshot()
		tier1Snaps := h.Qobuz.Tier1ProxyPoolSnapshot()
		tier1, failureKind, retryETASecs := poolSnapshotTier1Status(snaps, tier1Snaps, h.Qobuz.ProxyNextRevivalETASecs)
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

	if h.Amazon != nil {
		snaps := h.Amazon.PoolSnapshot()
		tier1Snaps := h.Amazon.Tier1PoolSnapshot()
		tier1, failureKind, retryETASecs := poolSnapshotTier1Status(snaps, tier1Snaps, h.Amazon.NextRevivalETASecs)
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

	results = append(results, core.SoulseekHealth(h.Soulseek, h.Config != nil && h.Config.SoulseekEnabled))

	return results
}

// poolSnapshotStatus aggregates the endpoints whose result is known: all OK is
// online, a mix degraded, all failing dead, none known untested. A failed last
// probe counts as failing even for a live endpoint; an untested endpoint
// without a failed probe is unknown and counts for nothing.
func poolSnapshotStatus(snaps []core.EndpointStat) string {
	ok, bad := 0, 0
	for _, ep := range snaps {
		switch {
		case ep.Probe != nil && ep.Probe.Kind == "failed", ep.State == "dead", ep.State == "blacklisted":
			bad++
		case ep.State == "live", ep.State == "probation":
			ok++
		}
	}
	switch {
	case ok+bad == 0:
		return "untested"
	case bad == 0:
		return "online"
	case ok == 0:
		return "dead"
	default:
		return "degraded"
	}
}

// poolSnapshotTier1Status derives a source's Tier1 status, FailureKind, and
// RetryETASecs purely from already-fetched pool snapshots, no network
// calls (see GetSourceHealth's doc comment). tier1Snaps empty means Tier1
// stays nil (not configured). A tier1 entry is healthy when neither dead nor
// untested, and Tested is false while every tier1 entry is untested.
// FailureKindUpstream is only reported once no tier1 entry is healthy AND
// every entry in the full snapshot (both tiers) is dead (untested entries do
// not count as dead), the same
// no-live-endpoint condition Core's own live probes (probeQobuz, probeAmazon,
// ProbeTidalService) use, just computed from a snapshot instead of a fresh
// check. An empty snaps means no endpoint is configured at all rather than
// every endpoint being down, so it reports FailureKindNone, matching the
// "untested" that poolSnapshotStatus returns for the same input into the
// neighbouring Status field. nextRevivalETASecs is only invoked when
// FailureKindUpstream is reported, since RetryETASecs is meaningless otherwise.
func poolSnapshotTier1Status(snaps, tier1Snaps []core.EndpointStat, nextRevivalETASecs func() int) (*core.Tier1Status, core.FailureKind, int) {
	var tier1 *core.Tier1Status
	tier1Healthy := false
	if len(tier1Snaps) > 0 {
		tier1Tested := false
		for _, ep := range tier1Snaps {
			if ep.State != "untested" {
				tier1Tested = true
				if ep.State != "dead" {
					tier1Healthy = true
				}
			}
		}
		tier1 = &core.Tier1Status{
			Configured: true,
			Healthy:    tier1Healthy,
			Tested:     tier1Tested,
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

	if !tier1Healthy && allDead && len(snaps) > 0 {
		return tier1, core.FailureKindUpstream, nextRevivalETASecs()
	}
	return tier1, core.FailureKindNone, 0
}

// TestSoulseekConnection logs in once with the given credentials through
// Core's built-in client, without running any external process.
func (a *App) TestSoulseekConnection(username, password string) map[string]interface{} {
	if a.core == nil {
		return SoulseekLoginResultMap(core.SoulseekLoginResult{Message: "core not initialised"})
	}
	res, err := a.core.TestSoulseekLogin(context.Background(), username, password)
	if err != nil {
		res = core.SoulseekLoginResult{Username: username, Message: err.Error()}
	}
	return SoulseekLoginResultMap(res)
}

// SoulseekLoginResultMap is the wire shape shared by the desktop (Wails) and
// HTTP server APIs for a Soulseek connection test.
func SoulseekLoginResultMap(res core.SoulseekLoginResult) map[string]interface{} {
	return map[string]interface{}{
		"success":  res.Success,
		"message":  res.Message,
		"username": res.Username,
	}
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
// has no native parser for, Spotify already has one, so this covers Apple
// Music, YouTube Music, Deezer short links, and similar, and returns the
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

func (a *App) discographyDeps() DiscographyDeps {
	return DiscographyDeps{
		Spotify:    a.spotifySearch,
		Tidal:      a.tidalSource,
		Downloader: a.downloader,
		Queue:      a.downloadManager,
		Log:        a.logBuffer,
	}
}

func (a *App) ExpandDiscographyURL(rawURL string) ([]string, error) {
	return ExpandDiscography(a.discographyDeps(), rawURL)
}

func (a *App) QueueDiscographyAlbums(spotifyAlbumURLs []string, outputDir string) (int, error) {
	return QueueDiscography(a.discographyDeps(), spotifyAlbumURLs, outputDir)
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

	next := a.updateConfig(func(c *core.Config) {
		c.QobuzAppID = appID
		c.QobuzAppSecret = appSecret
		c.QobuzAuthToken = authToken
		c.QobuzEnabled = appID != "" && appSecret != ""
	})

	if err := core.SaveConfig(&next); err != nil {
		return err
	}

	if a.logBuffer != nil {
		if next.QobuzEnabled {
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
