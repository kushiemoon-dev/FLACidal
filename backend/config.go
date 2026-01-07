package backend

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the application configuration
type Config struct {
	// Tidal credentials (optional - uses internal by default)
	TidalClientID     string `json:"tidalClientId,omitempty"`
	TidalClientSecret string `json:"tidalClientSecret,omitempty"`

	// Download settings
	DownloadFolder      string `json:"downloadFolder,omitempty"`
	DownloadQuality     string `json:"downloadQuality,omitempty"`     // "HI_RES", "LOSSLESS", "HIGH"
	FileNameFormat      string `json:"fileNameFormat,omitempty"`      // "{artist} - {title}", "{track} - {title}", etc.
	OrganizeFolders     bool   `json:"organizeFolders,omitempty"`     // Create Artist/Album/ subfolders
	EmbedCover          bool   `json:"embedCover"`                    // Embed cover art in FLAC
	ConcurrentDownloads int    `json:"concurrentDownloads,omitempty"` // Number of parallel downloads

	// UI settings
	Theme       string `json:"theme"`                 // "dark", "light", "system"
	AccentColor string `json:"accentColor,omitempty"` // Hex color e.g. "#f472b6"

	// Sound settings
	SoundEffects bool `json:"soundEffects"` // Enable/disable sound effects
	SoundVolume  int  `json:"soundVolume"`  // 0-100

	// Lyrics settings
	EmbedLyrics        bool `json:"embedLyrics"`        // Automatically fetch and embed lyrics
	PreferSyncedLyrics bool `json:"preferSyncedLyrics"` // Prefer synced (LRC) lyrics when available

	// Source settings
	TidalEnabled    bool   `json:"tidalEnabled"`              // Enable Tidal source
	QobuzEnabled    bool   `json:"qobuzEnabled"`              // Enable Qobuz source
	QobuzAppID      string `json:"qobuzAppId,omitempty"`      // Qobuz app ID
	QobuzAppSecret  string `json:"qobuzAppSecret,omitempty"`  // Qobuz app secret
	QobuzAuthToken  string `json:"qobuzAuthToken,omitempty"`  // Qobuz user auth token
	PreferredSource string `json:"preferredSource,omitempty"` // "tidal" or "qobuz"
}

var defaultConfig = Config{
	Theme:               "dark",
	AccentColor:         "#f472b6", // Pink (default)
	DownloadQuality:     "LOSSLESS",
	FileNameFormat:      "{artist} - {title}",
	OrganizeFolders:     false,
	EmbedCover:          true,
	ConcurrentDownloads: 4,
	SoundEffects:        false,
	SoundVolume:         70,
	EmbedLyrics:         false,
	PreferSyncedLyrics:  true,
	TidalEnabled:        true,
	QobuzEnabled:        false,
	PreferredSource:     "tidal",
}

// GetDataDir returns the app data directory (~/.flacidal/)
func GetDataDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".flacidal")
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	return filepath.Join(GetDataDir(), "config.json")
}

// GetDatabasePath returns the path to the SQLite database
func GetDatabasePath() string {
	return filepath.Join(GetDataDir(), "data.db")
}

// EnsureDataDir creates the data directory if it doesn't exist
func EnsureDataDir() error {
	return os.MkdirAll(GetDataDir(), 0755)
}

// LoadConfig loads configuration from file
func LoadConfig() (*Config, error) {
	if err := EnsureDataDir(); err != nil {
		return nil, err
	}

	configPath := GetConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			cfg := defaultConfig
			return &cfg, nil
		}
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config) error {
	if err := EnsureDataDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(GetConfigPath(), data, 0644)
}

// IsTidalConfigured checks if Tidal client credentials are configured
func (c *Config) IsTidalConfigured() bool {
	return c.TidalClientID != "" && c.TidalClientSecret != ""
}

// GetDefaultConfig returns a copy of the default configuration
func GetDefaultConfig() *Config {
	cfg := defaultConfig
	return &cfg
}

// GetDefaultDownloadFolder returns the default download directory
func GetDefaultDownloadFolder() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "Music", "FLACidal")
}

// LoadConfigWithEnv loads configuration from file with environment variable overrides
func LoadConfigWithEnv() (*Config, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	// Override with environment variables if set
	if val := os.Getenv("DOWNLOAD_FOLDER"); val != "" {
		config.DownloadFolder = val
	}
	if val := os.Getenv("DOWNLOAD_QUALITY"); val != "" {
		config.DownloadQuality = val
	}
	if val := os.Getenv("CONCURRENT_DOWNLOADS"); val != "" {
		if n, err := parseInt(val); err == nil && n > 0 {
			config.ConcurrentDownloads = n
		}
	}
	if val := os.Getenv("EMBED_COVER"); val != "" {
		config.EmbedCover = val == "true" || val == "1"
	}
	if val := os.Getenv("EMBED_LYRICS"); val != "" {
		config.EmbedLyrics = val == "true" || val == "1"
	}
	if val := os.Getenv("THEME"); val != "" {
		config.Theme = val
	}
	if val := os.Getenv("TIDAL_ENABLED"); val != "" {
		config.TidalEnabled = val == "true" || val == "1"
	}
	if val := os.Getenv("QOBUZ_ENABLED"); val != "" {
		config.QobuzEnabled = val == "true" || val == "1"
	}
	if val := os.Getenv("QOBUZ_APP_ID"); val != "" {
		config.QobuzAppID = val
	}
	if val := os.Getenv("QOBUZ_APP_SECRET"); val != "" {
		config.QobuzAppSecret = val
	}
	if val := os.Getenv("QOBUZ_AUTH_TOKEN"); val != "" {
		config.QobuzAuthToken = val
	}
	if val := os.Getenv("PREFERRED_SOURCE"); val != "" {
		config.PreferredSource = val
	}

	return config, nil
}

// parseInt helper for environment variable parsing
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
