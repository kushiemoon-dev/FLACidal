package app

import (
	"errors"

	core "github.com/kushiemoon-dev/flacidal-core"
)

var errNoCore = errors.New("core not initialised")

// The methods below expose Core's extension management to the Wails frontend.
// They are thin on purpose: Core owns validation, persistence and the
// resynchronisation of the download chain. The headless server mirrors them
// under /api/extensions/*.

func (a *App) ListExtensions() ([]core.ExtensionInfo, error) {
	if a.core == nil {
		return nil, errNoCore
	}
	return a.core.ListExtensions(), nil
}

func (a *App) InstallExtension(zipURL string) (*core.ExtensionManifest, error) {
	if a.core == nil {
		return nil, errNoCore
	}
	return a.core.InstallExtension(zipURL)
}

func (a *App) InstallExtensionFromRegistry(repoURL, id string) (*core.ExtensionManifest, error) {
	if a.core == nil {
		return nil, errNoCore
	}
	return a.core.InstallFromRegistry(repoURL, id)
}

func (a *App) UninstallExtension(id string) error {
	if a.core == nil {
		return errNoCore
	}
	return a.core.UninstallExtension(id)
}

func (a *App) SetExtensionEnabled(id string, enabled bool) error {
	if a.core == nil {
		return errNoCore
	}
	return a.core.SetExtensionEnabled(id, enabled)
}

func (a *App) SetExtensionAuth(id string, data map[string]string) error {
	if a.core == nil {
		return errNoCore
	}
	return a.core.SetExtensionAuth(id, data)
}

func (a *App) GetRegistrySources() ([]string, error) {
	if a.core == nil {
		return nil, errNoCore
	}
	return a.core.RegistrySources(), nil
}

func (a *App) AddRegistrySource(repoURL string) error {
	if a.core == nil {
		return errNoCore
	}
	return a.core.AddRegistrySource(repoURL)
}

func (a *App) RemoveRegistrySource(repoURL string) error {
	if a.core == nil {
		return errNoCore
	}
	return a.core.RemoveRegistrySource(repoURL)
}

func (a *App) BrowseRegistry(repoURL string) ([]core.RegistryEntry, error) {
	if a.core == nil {
		return nil, errNoCore
	}
	return a.core.BrowseRegistry(repoURL)
}

// GetEffectiveSourceOrder is the chain actually tried, extensions included.
func (a *App) GetEffectiveSourceOrder() ([]string, error) {
	if a.core == nil {
		return nil, errNoCore
	}
	return a.core.EffectiveSourceOrder(), nil
}
