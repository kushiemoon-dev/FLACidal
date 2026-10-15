package api

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestConfine(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	extra := t.TempDir()

	mustWrite := func(p string) string {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	inside := mustWrite(filepath.Join(root, "a.flac"))
	secret := mustWrite(filepath.Join(outside, "secret.flac"))
	extraFile := mustWrite(filepath.Join(extra, "lib.flac"))
	mustWrite(filepath.Join(root, "real", "b.flac"))
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	sibling := mustWrite(root + "x/file.flac")
	t.Cleanup(func() { os.RemoveAll(root + "x") })

	s := newTestServer(t)
	s.config.DownloadFolder = root
	s.config.ExternalLibraryPaths = []string{extra, ""}

	tests := []struct {
		name string
		path string
		want error
	}{
		{"inside root", inside, nil},
		{"internal symlink", filepath.Join(root, "alias", "b.flac"), nil},
		{"missing target under root", filepath.Join(root, "new", "deep", "c.flac"), nil},
		{"external library path", extraFile, nil},
		{"symlink then dotdot", filepath.Join(root, "escape") + "/../x.flac", errPathOutside},
		{"dotdot escape", filepath.Join(root, "..", filepath.Base(outside), "secret.flac"), errPathOutside},
		{"absolute outside", secret, errPathOutside},
		{"symlink out of root", filepath.Join(root, "escape", "secret.flac"), errPathOutside},
		{"missing target outside root", filepath.Join(outside, "nope", "x.flac"), errPathOutside},
		{"missing target via escaping symlink", filepath.Join(root, "escape", "nope", "x.flac"), errPathOutside},
		{"common prefix sibling", sibling, errPathOutside},
		{"relative", "a.flac", errPathInvalid},
		{"empty", "", errPathInvalid},
		{"null byte", root + "/a\x00.flac", errPathInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.confine(tt.path)
			if !errors.Is(err, tt.want) {
				t.Errorf("confine(%q) = %v, want %v", tt.path, err, tt.want)
			}
		})
	}
}

func TestConfineDefaultRoot(t *testing.T) {
	s := newTestServer(t)
	s.config.DownloadFolder = ""
	if err := s.confine("/definitely/not/the/default/x.flac"); !errors.Is(err, errPathOutside) {
		t.Errorf("confine outside default root = %v, want errPathOutside", err)
	}
}

func TestValidateDownloadFolder(t *testing.T) {
	for _, bad := range []string{"relative/dir", "/", "//", "/tmp/../", "."} {
		if _, err := validateDownloadFolder(bad); err == nil {
			t.Errorf("validateDownloadFolder(%q) = nil, want error", bad)
		}
	}
	if got, err := validateDownloadFolder(""); err != nil || got != "" {
		t.Errorf("empty = %q, %v; want \"\", nil", got, err)
	}
	dir := t.TempDir()
	if got, err := validateDownloadFolder(dir + "/"); err != nil || got != dir {
		t.Errorf("clean = %q, %v; want %s, nil", got, err, dir)
	}
}

func TestValidateLibraryPaths(t *testing.T) {
	for _, bad := range [][]string{{"/"}, {"rel"}, {""}, {"/ok", "/a/.."}} {
		if _, err := validateLibraryPaths(bad); err == nil {
			t.Errorf("validateLibraryPaths(%q) = nil, want error", bad)
		}
	}
	got, err := validateLibraryPaths([]string{"/srv/a/", "/srv/b"})
	if err != nil || len(got) != 2 || got[0] != "/srv/a" {
		t.Errorf("got %v, %v", got, err)
	}
	if got, err := validateLibraryPaths(nil); err != nil || len(got) != 0 {
		t.Errorf("nil = %v, %v", got, err)
	}
}
