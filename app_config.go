package main

import (
	"fmt"
	"os/exec"
	goruntime "runtime"
	"strings"

	core "github.com/kushiemoon-dev/flacidal-core"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// =============================================================================
// Config Methods (exposed to frontend)
// =============================================================================

// GetConfig returns current configuration
func (a *App) GetConfig() *core.Config {
	return a.config
}

// SaveConfig saves configuration
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
		// Re-apply priority endpoints live without restart
		if len(config.TidalHifiEndpoints) == 0 {
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
	if a.downloadManager != nil {
		a.downloadManager.SetSourceOrder(config.SourceOrder)
	}
	// Apply proxy changes immediately (no restart needed)
	if a.tidalClient != nil {
		if err := a.tidalClient.SetProxy(config.ProxyURL); err != nil {
			a.logBuffer.Warn("Proxy config error (Tidal API): " + err.Error())
		}
	}
	if a.downloader != nil {
		if err := a.downloader.SetProxy(config.ProxyURL); err != nil {
			a.logBuffer.Warn("Proxy config error (downloader): " + err.Error())
		}
	}
	if a.qobuzSource != nil {
		if err := a.qobuzSource.SetProxy(config.ProxyURL); err != nil {
			a.logBuffer.Warn("Proxy config error (Qobuz): " + err.Error())
		}
	}
	// Re-initialize Soulseek source when credentials or enabled state change
	sldlPath := config.SoulseekBinaryPath
	if sldlPath == "" {
		sldlPath = defaultSldlPath()
	}
	if err := ensureSldlExecutable(sldlPath); err != nil && a.logBuffer != nil {
		a.logBuffer.Warn(fmt.Sprintf("sldl binary may not be executable: %v", err))
	}
	a.soulseekSource = core.NewSoulseekSource(sldlPath, config.SoulseekUsername, config.SoulseekPassword)
	a.soulseekSource.SetLogger(a.logBuffer)
	if a.sourceManager != nil {
		if config.SoulseekEnabled && a.soulseekSource.IsAvailable() {
			a.sourceManager.RegisterSource(a.soulseekSource)
			if a.logBuffer != nil {
				a.logBuffer.Info("Soulseek fallback source registered")
			}
		} else {
			a.sourceManager.UnregisterSource("soulseek")
			if config.SoulseekEnabled && a.logBuffer != nil {
				a.logBuffer.Warn("Soulseek enabled but unavailable (check binary path / credentials)")
			}
		}
	}

	return core.SaveConfig(&config)
}

// SetSourceOrder updates the download source priority order live and persists it
func (a *App) SetSourceOrder(order []string) error {
	if len(order) == 0 {
		return fmt.Errorf("source order cannot be empty")
	}
	validSources := map[string]bool{"tidal": true, "qobuz": true, "amazon": true, "bandcamp": true, "soulseek": true}
	seen := map[string]bool{}
	for _, s := range order {
		if !validSources[s] {
			return fmt.Errorf("unknown source: %s", s)
		}
		if seen[s] {
			return fmt.Errorf("duplicate source: %s", s)
		}
		seen[s] = true
	}
	if a.orchestrator != nil {
		a.orchestrator.SetPriority(order)
	}
	if a.downloadManager != nil {
		a.downloadManager.SetSourceOrder(order)
	}
	if a.config != nil {
		a.config.SourceOrder = order
		return core.SaveConfig(a.config)
	}
	return nil
}

// ResetToDefaults resets configuration to default values
func (a *App) ResetToDefaults() (*core.Config, error) {
	defaultCfg := core.GetDefaultConfig()

	// Preserve download folder if set
	if a.config != nil && a.config.DownloadFolder != "" {
		defaultCfg.DownloadFolder = a.config.DownloadFolder
	}

	a.config = defaultCfg
	if err := core.SaveConfig(defaultCfg); err != nil {
		return nil, err
	}

	if a.logBuffer != nil {
		a.logBuffer.Info("Configuration reset to defaults")
		runtime.EventsEmit(a.ctx, "log", a.logBuffer.Info("Settings restored to defaults"))
	}

	return defaultCfg, nil
}

// GetConnectionStatus returns service status
func (a *App) GetConnectionStatus() map[string]interface{} {
	return map[string]interface{}{
		"tidalReady":    true, // Always ready (uses internal credentials)
		"spotifySearch": a.spotifySearch != nil,
	}
}

// EndpointStatus represents the status of an API endpoint
type EndpointStatus struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Status    string `json:"status"`    // "online", "offline", "slow"
	LatencyMs int64  `json:"latencyMs"` // Response time in milliseconds
}

// CheckAPIStatus returns the current pool state of all proxy endpoints without
// making any network requests. It reads from the in-memory EndpointPool snapshots
// maintained by the downloader and source services.
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

// OpenConfigFolder opens the app config directory in the system file manager
func (a *App) OpenConfigFolder() error {
	configDir := core.GetDataDir()
	return openFolder(configDir)
}

// openFolder opens a folder in the system file manager
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
