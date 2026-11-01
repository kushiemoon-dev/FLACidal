package app

import (
	"encoding/json"
	"errors"
	"fmt"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// ErrNotConfigExport marks a file that is not a FLACidal config export.
var ErrNotConfigExport = errors.New("not a FLACidal config export")

// secretFields lists the 7 credentials blanked on export and kept on import
// when the file leaves them empty. Single list so both sides never diverge.
func secretFields(c *core.Config) []*string {
	return []*string{
		&c.TidalClientID, &c.TidalClientSecret,
		&c.SoulseekUsername, &c.SoulseekPassword,
		&c.QobuzAppID, &c.QobuzAppSecret, &c.QobuzAuthToken,
	}
}

// BuildConfigExport serialises cfg as a schemaVersion 1 export with every
// secret blanked. cfg is passed by value, so the caller's config is untouched.
func BuildConfigExport(cfg core.Config) ([]byte, error) {
	for _, f := range secretFields(&cfg) {
		*f = ""
	}
	data, err := json.MarshalIndent(configExport{SchemaVersion: 1, Config: cfg}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("could not marshal config: %w", err)
	}
	return data, nil
}

// MergeImportedConfig parses an export and returns the config to apply: the
// file's values, except that an empty secret keeps the one in current.
func MergeImportedConfig(data []byte, current core.Config) (core.Config, error) {
	var export configExport
	if err := json.Unmarshal(data, &export); err != nil {
		return core.Config{}, fmt.Errorf("could not parse import file: %w", ErrNotConfigExport)
	}
	if export.SchemaVersion != 1 {
		return core.Config{}, fmt.Errorf("not a FLACidal config export (schemaVersion %d): %w", export.SchemaVersion, ErrNotConfigExport)
	}
	merged := export.Config
	cur := secretFields(&current)
	for i, f := range secretFields(&merged) {
		if *f == "" {
			*f = *cur[i]
		}
	}
	return merged, nil
}
