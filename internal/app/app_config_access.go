package app

import core "github.com/kushiemoon-dev/flacidal-core"

// a.config is replaced by SaveConfig and friends while queue workers and RPC
// handlers read it, so every access goes through these helpers. The stored
// Config is never mutated in place: readers get a copy and writers publish a
// new one.

// cfg returns a copy of the current config (zero value before startup).
func (a *App) cfg() core.Config {
	a.configMu.RLock()
	defer a.configMu.RUnlock()
	if a.config == nil {
		return core.Config{}
	}
	return *a.config
}

// configLoaded reports whether a config has been set.
func (a *App) configLoaded() bool {
	a.configMu.RLock()
	defer a.configMu.RUnlock()
	return a.config != nil
}

// setCfg replaces the config.
func (a *App) setCfg(c core.Config) {
	a.configMu.Lock()
	defer a.configMu.Unlock()
	a.config = &c
}

// updateConfig applies mutate to a copy of the config, publishes it and
// returns it, so callers can persist exactly what was published.
func (a *App) updateConfig(mutate func(*core.Config)) core.Config {
	a.configMu.Lock()
	defer a.configMu.Unlock()
	next := core.Config{}
	if a.config != nil {
		next = *a.config
	}
	mutate(&next)
	a.config = &next
	return next
}
