package app

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	goruntime "runtime"
	"strings"

	core "github.com/kushiemoon-dev/flacidal-core"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// configExport mirrors FLACidal-Core/rpc.go's configExport (kept independent
// per the D5 desktop/mobile split: desktop has no Core dependency), so a file
// exported by one side can be imported by the other.
type configExport struct {
	SchemaVersion int         `json:"schemaVersion"`
	Config        core.Config `json:"config"`
}

func (a *App) GetConfig() *core.Config {
	c := a.cfg()
	return &c
}

func (a *App) SaveConfig(config core.Config) error {
	// Only a changed folder is checked: the front end sends the whole config
	// back, and a vanished folder must not block saving other settings.
	if config.DownloadFolder != a.cfg().DownloadFolder {
		folder, err := core.ValidateDownloadFolder(config.DownloadFolder)
		if err != nil {
			return err
		}
		config.DownloadFolder = folder
	}
	if a.core == nil {
		a.setCfg(config)
		return core.SaveConfig(&config)
	}
	if err := a.core.ApplyConfig(config); err != nil {
		return err
	}
	// Rebuild the standalone sldl handle so its status follows a changed binary path or credentials.
	a.soulseekSource = NewSldlSource(&config, a.logBuffer)
	return nil
}

func (a *App) SetSourceOrder(order []string) error {
	validated, err := ValidateSourceOrder(order)
	if err != nil {
		return err
	}
	if a.core != nil {
		_, err := a.core.SetSourceOrder(validated)
		return err
	}
	if a.configLoaded() {
		next := a.updateConfig(func(c *core.Config) { c.SourceOrder = validated })
		return core.SaveConfig(&next)
	}
	return nil
}

// ValidateSourceOrder checks a submitted source order and rejects the entire
// call if it contains anything unknown or duplicated, the same behavior
// App.SetSourceOrder and the REST handler already rely on (this differs from
// FLACidal-Core's rpc_sources.go, which silently filters instead of
// rejecting). If the validated order is missing "soulseek", it gets
// appended last, since that's treated as a migration gap rather than an
// intentional omission (see core.NormalizeSourceOrder). An order that already
// has "soulseek" somewhere in it, first included, is passed through unchanged.
// This function backs both the desktop (Wails) and HTTP server APIs, the same
// sharing pattern used above by SldlStatus / TestSoulseekLogin.
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
	return core.NormalizeSourceOrder(order), nil
}

func (a *App) ResetToDefaults() (*core.Config, error) {
	defaultCfg := core.GetDefaultConfig()

	if folder := a.cfg().DownloadFolder; folder != "" {
		defaultCfg.DownloadFolder = folder
	}

	if a.core != nil {
		if err := a.core.ApplyConfig(*defaultCfg); err != nil {
			return nil, err
		}
		applied := a.core.Config()
		defaultCfg = &applied
	} else {
		a.setCfg(*defaultCfg)
		if err := core.SaveConfig(defaultCfg); err != nil {
			return nil, err
		}
	}

	if a.logBuffer != nil {
		a.logBuffer.Info("Settings reverted to defaults")
		runtime.EventsEmit(a.ctx, "log", a.logBuffer.Info("Defaults restored"))
	}

	return defaultCfg, nil
}

func (a *App) GetConnectionStatus() map[string]interface{} {
	return map[string]interface{}{
		"spotifySearch": a.spotifySearch != nil,
	}
}

type EndpointStatus struct {
	Name              string `json:"name"`
	URL               string `json:"url"`
	Status            string `json:"status"`                      // one of "online", "offline", "slow" or "untested"
	LatencyMs         int64  `json:"latencyMs"`                   // round-trip time, in milliseconds
	Message           string `json:"message,omitempty"`           // raw failure text of the last probe
	ReferenceNotFound bool   `json:"referenceNotFound,omitempty"` // every reference track was missing from this endpoint
}

// Reads from in-memory EndpointPool snapshots, issues no network calls itself.
func (a *App) CheckAPIStatus() []EndpointStatus {
	return BuildEndpointStatuses(a.healthSources())
}

// BuildEndpointStatuses lists the endpoints of every non-nil source from
// in-memory pool snapshots.
func BuildEndpointStatuses(h HealthSources) []EndpointStatus {
	var results []EndpointStatus

	if h.Downloader != nil {
		for _, ep := range h.Downloader.PoolSnapshot() {
			results = append(results, endpointStatToStatus("Tidal HiFi", ep))
		}
	}
	if h.Qobuz != nil {
		for _, ep := range h.Qobuz.ProxyPoolSnapshot() {
			results = append(results, endpointStatToStatus("Qobuz", ep))
		}
	}
	if h.Amazon != nil {
		for _, ep := range h.Amazon.PoolSnapshot() {
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
	case "live":
		status = "online"
	default: // "untested", and any unknown state: never show green without proof
		status = "untested"
	}
	var message string
	var notFound bool
	if ep.Probe != nil {
		switch ep.Probe.Kind {
		case "failed":
			status = "offline"
			message = ep.Probe.Message
		case "notfound":
			notFound = true
		}
	}
	host := ep.URL
	if idx := strings.Index(ep.URL, "://"); idx >= 0 {
		host = ep.URL[idx+3:]
	}
	return EndpointStatus{
		Name:              sourceLabel + " — " + host,
		URL:               ep.URL,
		Status:            status,
		LatencyMs:         ep.LatencyMs,
		Message:           message,
		ReferenceNotFound: notFound,
	}
}

func (a *App) OpenConfigFolder() error {
	configDir := core.GetDataDir()
	return openFolder(configDir)
}

// ExportConfig writes the current config to a user-chosen JSON file, with
// Tidal/Soulseek/Qobuz secrets redacted (same 7 fields as FLACidal-Core's
// rpc.go handleExportConfig). Returns nil with no error if the user cancels
// the save dialog.
func (a *App) ExportConfig() error {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: "flacidal-config.json",
		Filters:         []runtime.FileFilter{{DisplayName: "JSON Files", Pattern: "*.json"}},
	})
	if err != nil || path == "" {
		return err
	}
	cfg := a.cfg()
	cfg.TidalClientID = ""
	cfg.TidalClientSecret = ""
	cfg.SoulseekUsername = ""
	cfg.SoulseekPassword = ""
	cfg.QobuzAppID = ""
	cfg.QobuzAppSecret = ""
	cfg.QobuzAuthToken = ""
	data, err := json.MarshalIndent(configExport{SchemaVersion: 1, Config: cfg}, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// ImportConfig reads a config previously written by ExportConfig (or its
// FLACidal-Core mobile counterpart) from a user-chosen JSON file and applies
// it via SaveConfig, so it benefits from the same runtime re-wiring
// (downloader, downloadManager, sources) SaveConfig already does. Returns nil
// with no error if the user cancels the open dialog.
func (a *App) ImportConfig() (*core.Config, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Filters: []runtime.FileFilter{{DisplayName: "JSON Files", Pattern: "*.json"}},
	})
	if err != nil || path == "" {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read import file: %w", err)
	}
	var export configExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("could not parse import file: %w", err)
	}
	if export.SchemaVersion != 1 {
		return nil, fmt.Errorf("not a FLACidal config export (schemaVersion %d)", export.SchemaVersion)
	}
	if err := a.SaveConfig(export.Config); err != nil {
		return nil, err
	}
	return a.GetConfig(), nil
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
