package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

type errBody struct {
	Error string `json:"error"`
}

func folderServer(t *testing.T, current string) *Server {
	t.Helper()
	core.SetDataDir(t.TempDir())
	t.Cleanup(func() { core.SetDataDir("") })
	return NewServer(ServerConfig{Config: &core.Config{DownloadFolder: current}})
}

func TestFolderValidation_Rejected(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct{ folder, code string }{
		"relative": {"rel/dir", "folder.relative"},
		"root":     {"/", "folder.root"},
		"missing":  {filepath.Join(dir, "absent"), "folder.missing"},
		"file":     {file, "folder.not_dir"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := folderServer(t, "")
			var e errBody
			resp := doRequest(t, s, "POST", "/api/folder", map[string]string{"folder": tc.folder}, &e)
			if resp.StatusCode != fiber.StatusBadRequest || !strings.Contains(e.Error, "["+tc.code+"]") {
				t.Errorf("/api/folder: status %d, error %q", resp.StatusCode, e.Error)
			}
			e = errBody{}
			resp = doRequest(t, s, "POST", "/api/config", map[string]string{"downloadFolder": tc.folder}, &e)
			if resp.StatusCode != fiber.StatusBadRequest || !strings.Contains(e.Error, "["+tc.code+"]") {
				t.Errorf("/api/config: status %d, error %q", resp.StatusCode, e.Error)
			}
		})
	}
}

func TestFolderValidation_Accepted(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	for name, folder := range map[string]string{"valid": dir, "symlink": link, "empty": ""} {
		t.Run(name, func(t *testing.T) {
			s := folderServer(t, "")
			if resp := doRequest(t, s, "POST", "/api/folder", map[string]string{"folder": folder}, nil); resp.StatusCode != fiber.StatusOK {
				t.Errorf("/api/folder status = %d", resp.StatusCode)
			}
			if resp := doRequest(t, s, "POST", "/api/config", map[string]string{"downloadFolder": folder}, nil); resp.StatusCode != fiber.StatusOK {
				t.Errorf("/api/config status = %d", resp.StatusCode)
			}
		})
	}
}

// The front end sends the whole config back, so a vanished (unchanged)
// folder must not block saving unrelated settings.
func TestSaveConfig_UnchangedMissingFolderStillSaves(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "unmounted")
	s := folderServer(t, gone)
	body := map[string]string{"downloadFolder": gone, "theme": "light"}
	if resp := doRequest(t, s, "POST", "/api/config", body, nil); resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if s.cfg().Theme != "light" {
		t.Errorf("theme not saved")
	}
}

func TestListFiles_ConfiguredFolderStates(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent")
	var e errBody
	resp := doRequest(t, folderServer(t, missing), "GET", "/api/files", nil, &e)
	if resp.StatusCode != fiber.StatusInternalServerError || !strings.Contains(e.Error, "[folder.missing]") {
		t.Errorf("missing: status %d, error %q", resp.StatusCode, e.Error)
	}

	var files []core.DownloadedFileInfo
	resp = doRequest(t, folderServer(t, t.TempDir()), "GET", "/api/files", nil, &files)
	if resp.StatusCode != fiber.StatusOK || files == nil || len(files) != 0 {
		t.Errorf("empty: status %d, files %#v", resp.StatusCode, files)
	}

	s := folderServer(t, "")
	_ = s.updateConfig(func(c *core.Config) { c.DownloadFolder = t.TempDir() })
	var got struct {
		Folder string `json:"folder"`
	}
	doRequest(t, s, "GET", "/api/folder", nil, &got)
	if got.Folder != s.cfg().DownloadFolder || got.Folder == "" {
		t.Errorf("GET /api/folder = %q", got.Folder)
	}
}
