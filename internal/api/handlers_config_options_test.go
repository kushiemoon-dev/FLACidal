package api

import (
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Tests for the ResetConfig / DownloadOptions parity fixes: preserving the
// download folder on reset, and round-tripping saveFolderCover/autoAnalyze
// (previously silently dropped by these two handlers).

func TestHandleResetConfig_PreservesDownloadFolder(t *testing.T) {
	core.SetDataDir(t.TempDir())
	s := NewServer(ServerConfig{
		Config: &core.Config{DownloadFolder: "/music/downloads"},
	})

	var result core.Config
	resp := doRequest(t, s, "POST", "/api/config/reset", nil, &result)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if result.DownloadFolder != "/music/downloads" {
		t.Errorf("DownloadFolder = %q, want preserved %q", result.DownloadFolder, "/music/downloads")
	}
}

func TestHandleGetDownloadOptions_IncludesSaveFolderCoverAndAutoAnalyze(t *testing.T) {
	s := NewServer(ServerConfig{
		Config: &core.Config{SaveFolderCover: true, AutoAnalyze: true},
	})

	var body map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/downloads/options", nil, &body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if body["saveFolderCover"] != true {
		t.Errorf("saveFolderCover = %v, want true", body["saveFolderCover"])
	}
	if body["autoAnalyze"] != true {
		t.Errorf("autoAnalyze = %v, want true", body["autoAnalyze"])
	}
}

func TestHandleSetDownloadOptions_PersistsAutoAnalyze(t *testing.T) {
	core.SetDataDir(t.TempDir())
	s := NewServer(ServerConfig{Config: &core.Config{}})

	var body map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/downloads/options", map[string]interface{}{
		"quality":         "LOSSLESS",
		"fileNameFormat":  "{artist} - {title}",
		"organizeFolders": false,
		"embedCover":      true,
		"saveCoverFile":   true,
		"autoAnalyze":     true,
	}, &body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if !s.config.AutoAnalyze {
		t.Error("config.AutoAnalyze = false, want true to have been persisted")
	}
}
