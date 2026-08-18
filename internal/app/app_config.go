package app

import (
	"fmt"
	"os/exec"
	goruntime "runtime"
	"slices"
	"strings"

	core "github.com/kushiemoon-dev/flacidal-core"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) GetConfig() *core.Config {
	return a.config
}

func (a *App) SaveConfig(config core.Config) error {
	a.config = &config
	if a.downloadManager != nil {
		a.downloadManager.SetGenerateM3U8(config.GenerateM3U8)
		a.downloadManager.SetSkipUnavailable(config.SkipUnavailableTracks)
		a.downloadManager.SetJellyfin(config.JellyfinEnabled, config.JellyfinURL, config.JellyfinAPIKey)
	}
	if a.downloader != nil {
		opts := a.downloader.GetOptions()
		opts.AutoQualityFallback = config.AutoQualityFallback
		opts.QualityFallbackOrder = config.QualityOrder
		opts.FirstArtistOnly = config.FirstArtistOnly
		opts.SkipExisting = config.SkipExisting
		opts.ExternalLibraryPaths = config.ExternalLibraryPaths
		opts.ArtistSeparator = config.ArtistSeparator
		opts.PlaylistSubfolder = config.PlaylistSubfolder
		if config.DownloadQuality != "" {
			opts.Quality = config.DownloadQuality
		}
		if config.FileNameFormat != "" {
			opts.FileNameFormat = config.FileNameFormat
		}
		opts.OrganizeFolders = config.OrganizeFolders
		opts.FolderTemplate = config.FolderTemplate
		opts.EmbedCover = config.EmbedCover
		opts.SaveCoverFile = config.SaveCoverFile
		opts.AutoAnalyze = config.AutoAnalyze
		opts.SaveLyricsFile = config.SaveLyricsFile
		opts.SaveFolderCover = config.SaveFolderCover
		a.downloader.SetOptions(opts)
		// An override always wins, otherwise priority entries go ahead of the public pool.
		if len(config.TidalHifiEndpoints) > 0 {
			a.downloader.SetEndpoints(config.TidalHifiEndpoints)
		} else {
			base := core.GetTidalEndpoints()
			priority := config.TidalPriorityEndpoints
			if len(priority) == 0 && config.TidalCustomEndpoint != "" {
				priority = []string{config.TidalCustomEndpoint}
			}
			if len(priority) > 0 {
				a.downloader.SetEndpoints(append(priority, base...))
			} else {
				a.downloader.SetEndpoints(base)
			}
		}
	}
	if a.tidalSource != nil {
		// Copy the downloader's endpoint list onto the source manager's Tidal
		// instance too, since it's the one used for playlist/album/track fetch — see Startup for the equivalent wiring.
		if len(config.TidalHifiEndpoints) > 0 {
			a.tidalSource.GetService().SetEndpoints(config.TidalHifiEndpoints)
		} else {
			base := core.GetTidalEndpoints()
			priority := config.TidalPriorityEndpoints
			if len(priority) == 0 && config.TidalCustomEndpoint != "" {
				priority = []string{config.TidalCustomEndpoint}
			}
			if len(priority) > 0 {
				a.tidalSource.GetService().SetEndpoints(append(priority, base...))
			} else {
				a.tidalSource.GetService().SetEndpoints(base)
			}
		}
	}
	if a.qobuzSource != nil {
		// An override always wins, otherwise priority entries go ahead of the public pool.
		if len(config.QobuzEndpoints) > 0 {
			a.qobuzSource.SetEndpoints(config.QobuzEndpoints)
		} else {
			base := core.DefaultQobuzEndpoints()
			priority := config.QobuzPriorityEndpoints
			if len(priority) == 0 && config.QobuzCustomEndpoint != "" {
				priority = []string{config.QobuzCustomEndpoint}
			}
			if len(priority) > 0 {
				a.qobuzSource.SetEndpoints(append(priority, base...))
			} else {
				a.qobuzSource.SetEndpoints(base)
			}
		}
	}
	if a.amazonSource != nil {
		// Refresh endpoints on the fly, no restart required: an override always wins, otherwise priority entries go ahead of the public pool.
		if len(config.AmazonProxyEndpoints) > 0 {
			a.amazonSource.SetEndpoints(config.AmazonProxyEndpoints)
		} else {
			base := core.GetEndpoints("amazon")
			if priority := config.AmazonPriorityEndpoints; len(priority) > 0 {
				a.amazonSource.SetEndpoints(append(priority, base...))
			} else {
				a.amazonSource.SetEndpoints(base)
			}
		}
	}
	if a.downloadManager != nil {
		a.downloadManager.SetSourceOrder(config.SourceOrder)
	}
	// Proxy changes take effect right away, no restart required
	if a.tidalClient != nil {
		if err := a.tidalClient.SetProxy(config.ProxyURL); err != nil {
			a.logBuffer.Warn("Tidal API proxy misconfigured: " + err.Error())
		}
	}
	if a.downloader != nil {
		if err := a.downloader.SetProxy(config.ProxyURL); err != nil {
			a.logBuffer.Warn("Downloader proxy misconfigured: " + err.Error())
		}
	}
	if a.qobuzSource != nil {
		if err := a.qobuzSource.SetProxy(config.ProxyURL); err != nil {
			a.logBuffer.Warn("Qobuz proxy misconfigured: " + err.Error())
		}
	}
	sldlPath := config.SoulseekBinaryPath
	if sldlPath == "" {
		sldlPath = DefaultSldlPath()
	}
	if err := EnsureSldlExecutable(sldlPath); err != nil && a.logBuffer != nil {
		a.logBuffer.Warn(fmt.Sprintf("sldl binary might not be runnable: %v", err))
	}
	a.soulseekSource = core.NewSoulseekSource(sldlPath, config.SoulseekUsername, config.SoulseekPassword)
	a.soulseekSource.SetLogger(a.logBuffer)
	if a.sourceManager != nil {
		if config.SoulseekEnabled && a.soulseekSource.IsAvailable() {
			a.sourceManager.RegisterSource(a.soulseekSource)
			if a.logBuffer != nil {
				a.logBuffer.Info("Registered the Soulseek fallback source")
			}
		} else {
			a.sourceManager.UnregisterSource("soulseek")
			if config.SoulseekEnabled && a.logBuffer != nil {
				a.logBuffer.Warn("Soulseek is enabled but not usable — check its binary path and credentials")
			}
		}
	}

	return core.SaveConfig(&config)
}

func (a *App) SetSourceOrder(order []string) error {
	validated, err := ValidateSourceOrder(order)
	if err != nil {
		return err
	}
	if a.orchestrator != nil {
		a.orchestrator.SetPriority(validated)
	}
	if a.downloadManager != nil {
		a.downloadManager.SetSourceOrder(validated)
	}
	if a.config != nil {
		a.config.SourceOrder = validated
		return core.SaveConfig(a.config)
	}
	return nil
}

// ValidateSourceOrder checks a submitted source order and rejects the entire
// call if it contains anything unknown or duplicated — the same behavior
// App.SetSourceOrder and the REST handler already rely on (this differs from
// FLACidal-Core's rpc_sources.go, which silently filters instead of
// rejecting). If the validated order is missing "soulseek", it gets
// prepended as primary, since that's treated as a migration gap rather than
// an intentional omission (mirroring the migration in
// FLACidal-Core/core.go's NewCore() and rpc_sources.go's
// handleSetSourceOrder). An order that already has "soulseek" somewhere in
// it is passed through unmodified — the prepend only fires when it's
// absent. This function backs both the desktop (Wails) and HTTP server
// APIs, the same sharing pattern used above by SldlStatus / TestSoulseekLogin.
func ValidateSourceOrder(order []string) ([]string, error) {
	if len(order) == 0 {
		return nil, fmt.Errorf("source order cannot be empty")
	}
	validSources := map[string]bool{"tidal": true, "qobuz": true, "amazon": true, "bandcamp": true, "soulseek": true}
	seen := map[string]bool{}
	for _, s := range order {
		if !validSources[s] {
			return nil, fmt.Errorf("unknown source: %s", s)
		}
		if seen[s] {
			return nil, fmt.Errorf("duplicate source: %s", s)
		}
		seen[s] = true
	}
	if !slices.Contains(order, "soulseek") {
		order = append([]string{"soulseek"}, order...)
	}
	return order, nil
}

func (a *App) ResetToDefaults() (*core.Config, error) {
	defaultCfg := core.GetDefaultConfig()

	if a.config != nil && a.config.DownloadFolder != "" {
		defaultCfg.DownloadFolder = a.config.DownloadFolder
	}

	a.config = defaultCfg
	if err := core.SaveConfig(defaultCfg); err != nil {
		return nil, err
	}

	if a.logBuffer != nil {
		a.logBuffer.Info("Settings reverted to defaults")
		runtime.EventsEmit(a.ctx, "log", a.logBuffer.Info("Defaults restored"))
	}

	return defaultCfg, nil
}

func (a *App) GetConnectionStatus() map[string]interface{} {
	return map[string]interface{}{
		"tidalReady":    true, // built-in credentials mean this is always ready
		"spotifySearch": a.spotifySearch != nil,
	}
}

type EndpointStatus struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Status    string `json:"status"`    // one of "online", "offline", "slow"
	LatencyMs int64  `json:"latencyMs"` // round-trip time, in milliseconds
}

// Reads from in-memory EndpointPool snapshots — issues no network calls itself.
func (a *App) CheckAPIStatus() []EndpointStatus {
	var results []EndpointStatus

	if a.downloader != nil {
		for _, ep := range a.downloader.PoolSnapshot() {
			results = append(results, endpointStatToStatus("Tidal HiFi", ep))
		}
	}
	if a.qobuzSource != nil {
		for _, ep := range a.qobuzSource.ProxyPoolSnapshot() {
			results = append(results, endpointStatToStatus("Qobuz", ep))
		}
	}
	if a.amazonSource != nil {
		for _, ep := range a.amazonSource.PoolSnapshot() {
			results = append(results, endpointStatToStatus("Amazon", ep))
		}
	}

	return results
}

func endpointStatToStatus(sourceLabel string, ep core.EndpointStat) EndpointStatus {
	var status string
	switch ep.State {
	case "dead":
		status = "offline"
	case "blacklisted", "probation":
		status = "slow"
	default:
		status = "online"
	}
	host := ep.URL
	if idx := strings.Index(ep.URL, "://"); idx >= 0 {
		host = ep.URL[idx+3:]
	}
	return EndpointStatus{
		Name:      sourceLabel + " — " + host,
		URL:       ep.URL,
		Status:    status,
		LatencyMs: ep.LatencyMs,
	}
}

func (a *App) OpenConfigFolder() error {
	configDir := core.GetDataDir()
	return openFolder(configDir)
}

func openFolder(path string) error {
	switch goruntime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	case "windows":
		return exec.Command("explorer", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}
