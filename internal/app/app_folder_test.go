package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func TestSetDownloadFolder_Validation(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}

	rejected := map[string]string{
		"rel/dir":                    "folder.relative",
		"/":                          "folder.root",
		filepath.Join(dir, "absent"): "folder.missing",
		file:                         "folder.not_dir",
	}
	for folder, code := range rejected {
		a := newTestApp(t)
		a.config = &core.Config{DownloadFolder: "/keep"}
		err := a.SetDownloadFolder(folder)
		if err == nil || !strings.Contains(err.Error(), "["+code+"]") {
			t.Errorf("SetDownloadFolder(%q) = %v, want %s", folder, err, code)
		}
		if a.GetDownloadFolder() != "/keep" {
			t.Errorf("rejected folder %q changed the config", folder)
		}
	}

	for _, folder := range []string{dir, link, ""} {
		a := newTestApp(t)
		if err := a.SetDownloadFolder(folder); err != nil {
			t.Errorf("SetDownloadFolder(%q) = %v", folder, err)
		}
	}
}

func TestSaveConfig_FolderValidation(t *testing.T) {
	a := newTestApp(t)
	gone := filepath.Join(t.TempDir(), "unmounted")
	a.config = &core.Config{DownloadFolder: gone}

	if err := a.SaveConfig(core.Config{DownloadFolder: gone, Theme: "light"}); err != nil {
		t.Fatalf("unchanged missing folder must still save: %v", err)
	}
	if a.GetConfig().Theme != "light" {
		t.Error("theme not saved")
	}
	err := a.SaveConfig(core.Config{DownloadFolder: "rel"})
	if err == nil || !strings.Contains(err.Error(), "[folder.relative]") {
		t.Errorf("changed relative folder = %v, want folder.relative", err)
	}
}
