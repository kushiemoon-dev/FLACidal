package backend

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetDataDir(t *testing.T) {
	result := GetDataDir()
	if result == "" {
		t.Error("GetDataDir returned empty string")
	}

	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".flacidal")
	if result != expected {
		t.Errorf("GetDataDir() = %q, want %q", result, expected)
	}
}

func TestGetConfigPath(t *testing.T) {
	result := GetConfigPath()
	expected := filepath.Join(GetDataDir(), "config.json")
	if result != expected {
		t.Errorf("GetConfigPath() = %q, want %q", result, expected)
	}
}

func TestGetDatabasePath(t *testing.T) {
	result := GetDatabasePath()
	expected := filepath.Join(GetDataDir(), "data.db")
	if result != expected {
		t.Errorf("GetDatabasePath() = %q, want %q", result, expected)
	}
}

func TestGetDefaultConfig(t *testing.T) {
	cfg := GetDefaultConfig()

	if cfg == nil {
		t.Fatal("GetDefaultConfig returned nil")
	}

	if cfg.Theme != "dark" {
		t.Errorf("Default theme = %q, want 'dark'", cfg.Theme)
	}

	if cfg.DownloadQuality != "HI_RES" {
		t.Errorf("Default download quality = %q, want 'HI_RES'", cfg.DownloadQuality)
	}

	if cfg.FileNameFormat != "{artist} - {title}" {
		t.Errorf("Default filename format = %q, want '{artist} - {title}'", cfg.FileNameFormat)
	}

	if cfg.ConcurrentDownloads != 4 {
		t.Errorf("Default concurrent downloads = %d, want 4", cfg.ConcurrentDownloads)
	}

	if !cfg.EmbedCover {
		t.Error("Default EmbedCover should be true")
	}

	if !cfg.SaveCoverFile {
		t.Error("Default SaveCoverFile should be true")
	}
}

func TestGetDefaultDownloadFolder(t *testing.T) {
	result := GetDefaultDownloadFolder()
	if result == "" {
		t.Error("GetDefaultDownloadFolder returned empty string")
	}

	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, "Music", "FLACidal")
	if result != expected {
		t.Errorf("GetDefaultDownloadFolder() = %q, want %q", result, expected)
	}
}

func TestConfigIsTidalConfigured(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected bool
	}{
		{
			name: "configured",
			config: Config{
				TidalClientID:     "test-id",
				TidalClientSecret: "test-secret",
			},
			expected: true,
		},
		{
			name: "missing secret",
			config: Config{
				TidalClientID:     "test-id",
				TidalClientSecret: "",
			},
			expected: false,
		},
		{
			name: "missing id",
			config: Config{
				TidalClientID:     "",
				TidalClientSecret: "test-secret",
			},
			expected: false,
		},
		{
			name:     "not configured",
			config:   Config{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsTidalConfigured()
			if result != tt.expected {
				t.Errorf("IsTidalConfigured() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfigStructDefaults(t *testing.T) {
	cfg := &Config{}

	if cfg.Theme != "" {
		t.Errorf("Empty config Theme = %q, want empty", cfg.Theme)
	}

	if cfg.DownloadFolder != "" {
		t.Errorf("Empty config DownloadFolder = %q, want empty", cfg.DownloadFolder)
	}
}

func TestConfigStructAllFields(t *testing.T) {
	cfg := Config{
		TidalClientID:         "client-id",
		TidalClientSecret:     "client-secret",
		DownloadFolder:        "/downloads",
		DownloadQuality:       "HI_RES",
		FileNameFormat:        "{track} - {title}",
		OrganizeFolders:       true,
		EmbedCover:            false,
		SaveCoverFile:         false,
		ConcurrentDownloads:   8,
		Theme:                 "light",
		AccentColor:           "#00ff00",
		SoundEffects:          true,
		SoundVolume:           50,
		EmbedLyrics:           true,
		PreferSyncedLyrics:    false,
		AutoAnalyze:           true,
		AutoQualityFallback:   true,
		TidalEnabled:          true,
		QobuzEnabled:          true,
		QobuzAppID:            "qobuz-id",
		QobuzAppSecret:        "qobuz-secret",
		QobuzAuthToken:        "qobuz-token",
		PreferredSource:       "qobuz",
		TidalHifiEndpoints:    []string{"https://custom.endpoint"},
		QobuzEndpoints:        []string{"https://qobuz.endpoint"},
		SourceOrder:           []string{"qobuz", "tidal"},
		QualityOrder:          []string{"HI_RES", "LOSSLESS"},
		GenerateM3U8:          true,
		SkipUnavailableTracks: true,
		FirstArtistOnly:       true,
		ProxyURL:              "socks5://127.0.0.1:1080",
	}

	if cfg.TidalClientID != "client-id" {
		t.Errorf("TidalClientID = %q, want 'client-id'", cfg.TidalClientID)
	}
	if cfg.ConcurrentDownloads != 8 {
		t.Errorf("ConcurrentDownloads = %d, want 8", cfg.ConcurrentDownloads)
	}
	if len(cfg.TidalHifiEndpoints) != 1 {
		t.Errorf("TidalHifiEndpoints length = %d, want 1", len(cfg.TidalHifiEndpoints))
	}
}

func TestLoadConfigWithEnv(t *testing.T) {
	t.Run("DOWNLOAD_FOLDER override", func(t *testing.T) {
		os.Setenv("DOWNLOAD_FOLDER", "/custom/downloads")
		defer os.Unsetenv("DOWNLOAD_FOLDER")

		cfg, err := LoadConfigWithEnv()
		if err != nil {
			t.Fatalf("LoadConfigWithEnv failed: %v", err)
		}

		if cfg.DownloadFolder != "/custom/downloads" {
			t.Errorf("DownloadFolder = %q, want '/custom/downloads'", cfg.DownloadFolder)
		}
	})

	t.Run("DOWNLOAD_QUALITY override", func(t *testing.T) {
		os.Setenv("DOWNLOAD_QUALITY", "HI_RES")
		defer os.Unsetenv("DOWNLOAD_QUALITY")

		cfg, err := LoadConfigWithEnv()
		if err != nil {
			t.Fatalf("LoadConfigWithEnv failed: %v", err)
		}

		if cfg.DownloadQuality != "HI_RES" {
			t.Errorf("DownloadQuality = %q, want 'HI_RES'", cfg.DownloadQuality)
		}
	})

	t.Run("CONCURRENT_DOWNLOADS override", func(t *testing.T) {
		os.Setenv("CONCURRENT_DOWNLOADS", "16")
		defer os.Unsetenv("CONCURRENT_DOWNLOADS")

		cfg, err := LoadConfigWithEnv()
		if err != nil {
			t.Fatalf("LoadConfigWithEnv failed: %v", err)
		}

		if cfg.ConcurrentDownloads != 16 {
			t.Errorf("ConcurrentDownloads = %d, want 16", cfg.ConcurrentDownloads)
		}
	})

	t.Run("EMBED_COVER override", func(t *testing.T) {
		os.Setenv("EMBED_COVER", "false")
		defer os.Unsetenv("EMBED_COVER")

		cfg, err := LoadConfigWithEnv()
		if err != nil {
			t.Fatalf("LoadConfigWithEnv failed: %v", err)
		}

		if cfg.EmbedCover {
			t.Error("EmbedCover should be false")
		}
	})

	t.Run("EMBED_COVER true override", func(t *testing.T) {
		os.Setenv("EMBED_COVER", "true")
		defer os.Unsetenv("EMBED_COVER")

		cfg, err := LoadConfigWithEnv()
		if err != nil {
			t.Fatalf("LoadConfigWithEnv failed: %v", err)
		}

		if !cfg.EmbedCover {
			t.Error("EmbedCover should be true")
		}
	})

	t.Run("TIDAL_ENABLED override", func(t *testing.T) {
		os.Setenv("TIDAL_ENABLED", "false")
		defer os.Unsetenv("TIDAL_ENABLED")

		cfg, err := LoadConfigWithEnv()
		if err != nil {
			t.Fatalf("LoadConfigWithEnv failed: %v", err)
		}

		if cfg.TidalEnabled {
			t.Error("TidalEnabled should be false")
		}
	})

	t.Run("THEME override", func(t *testing.T) {
		os.Setenv("THEME", "light")
		defer os.Unsetenv("THEME")

		cfg, err := LoadConfigWithEnv()
		if err != nil {
			t.Fatalf("LoadConfigWithEnv failed: %v", err)
		}

		if cfg.Theme != "light" {
			t.Errorf("Theme = %q, want 'light'", cfg.Theme)
		}
	})
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		hasError bool
	}{
		{"123", 123, false},
		{"0", 0, false},
		{"-5", -5, false},
		{"", 0, true},
		{"abc", 0, true},
		{"12.5", 12, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseInt(tt.input)

			if tt.hasError {
				if err == nil {
					t.Error("Expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("parseInt(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}
