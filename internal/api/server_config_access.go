package api

import core "github.com/kushiemoon-dev/flacidal-core"

// s.config is replaced by the config handlers while other handlers read it,
// so every access goes through these helpers. The stored Config is never
// mutated in place: readers get a copy and writers publish a new one. When the
// server runs on a Core, Core owns the config and these helpers delegate to it.

// cfg returns a copy of the current config (zero value when none was set).
func (s *Server) cfg() core.Config {
	if s.core != nil {
		return s.core.Config()
	}
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	if s.config == nil {
		return core.Config{}
	}
	return *s.config
}

// configLoaded reports whether a config has been set.
func (s *Server) configLoaded() bool {
	if s.core != nil {
		return true
	}
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.config != nil
}

// setCfg replaces the config.
func (s *Server) setCfg(c core.Config) {
	if s.core != nil {
		s.core.UpdateConfig(func(cur *core.Config) { *cur = c })
		return
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.config = &c
}

// updateConfig applies mutate to a copy of the config, publishes it and
// returns it, so callers can persist exactly what was published.
func (s *Server) updateConfig(mutate func(*core.Config)) core.Config {
	if s.core != nil {
		return s.core.UpdateConfig(mutate)
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()
	next := core.Config{}
	if s.config != nil {
		next = *s.config
	}
	mutate(&next)
	s.config = &next
	return next
}
