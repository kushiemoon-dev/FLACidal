package api

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

type browseResp struct {
	Path      string `json:"path"`
	Parent    string `json:"parent"`
	Truncated bool   `json:"truncated"`
	Entries   []struct {
		Name  string `json:"name"`
		Path  string `json:"path"`
		IsDir bool   `json:"isDir"`
	} `json:"entries"`
}

func (b browseResp) names() []string {
	out := make([]string, 0, len(b.Entries))
	for _, e := range b.Entries {
		out = append(out, e.Name)
	}
	return out
}

func (b browseResp) has(name string) bool {
	for _, e := range b.Entries {
		if e.Name == name {
			return true
		}
	}
	return false
}

func browse(t *testing.T, s *Server, scope, path string, files bool) (int, browseResp) {
	t.Helper()
	q := url.Values{"scope": {scope}, "path": {path}}
	if files {
		q.Set("files", "1")
	}
	var out browseResp
	resp := doRequest(t, s, "GET", "/api/browse?"+q.Encode(), nil, &out)
	return resp.StatusCode, out
}

func touch(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBrowseToolsRootAndSubfolder(t *testing.T) {
	f := newGuardFixture(t)
	if err := os.Mkdir(filepath.Join(f.root, "Albums"), 0o755); err != nil {
		t.Fatal(err)
	}
	touch(t, filepath.Join(f.root, "Albums", "a.flac"))

	code, got := browse(t, f.s, "tools", "", false)
	if code != 200 || got.Path != f.root || !got.has("Albums") {
		t.Fatalf("empty path: %d %+v", code, got)
	}
	code, got = browse(t, f.s, "tools", filepath.Join(f.root, "Albums"), true)
	if code != 200 || !got.has("a.flac") || got.Parent != f.root {
		t.Fatalf("subfolder: %d %+v", code, got)
	}
}

func TestBrowseToolsParentEmptyAtRoot(t *testing.T) {
	f := newGuardFixture(t)
	_, got := browse(t, f.s, "tools", f.root, false)
	if got.Parent != "" {
		t.Errorf("parent of root = %q, want empty", got.Parent)
	}
}

func TestBrowseToolsRefusesEscapes(t *testing.T) {
	f := newGuardFixture(t)
	outDir := filepath.Dir(f.outside)
	for name, p := range map[string]string{
		"outside":  outDir,
		"etc":      "/etc",
		"dotdot":   f.root + "/../" + filepath.Base(outDir),
		"symlink":  filepath.Join(f.root, "outdir"),
		"absentoo": "/nonexistent-xyz",
	} {
		code, _ := browse(t, f.s, "tools", p, false)
		if code != 403 {
			t.Errorf("%s: status %d, want 403", name, code)
		}
	}
	if code, _ := browse(t, f.s, "tools", "relative/dir", false); code != 400 {
		t.Errorf("relative: status %d, want 400", code)
	}
}

func TestBrowseToolsOmitsOutgoingSymlinksAndHidden(t *testing.T) {
	f := newGuardFixture(t)
	touch(t, filepath.Join(f.root, ".hidden", "x.flac"))
	touch(t, filepath.Join(f.root, ".secret.flac"))
	_, got := browse(t, f.s, "tools", f.root, true)
	for _, bad := range []string{"outdir", "link.flac", ".hidden", ".secret.flac"} {
		if got.has(bad) {
			t.Errorf("%q must be omitted, got %v", bad, got.names())
		}
	}
}

func TestBrowseToolsFilesOnlyWhenAskedAndOnlyFlac(t *testing.T) {
	f := newGuardFixture(t)
	touch(t, filepath.Join(f.root, "a.flac"))
	touch(t, filepath.Join(f.root, "B.FLAC"))
	touch(t, filepath.Join(f.root, "c.mp3"))
	touch(t, filepath.Join(f.root, "d.txt"))

	_, off := browse(t, f.s, "tools", f.root, false)
	if len(off.Entries) != 0 {
		t.Errorf("files=0 listed %v", off.names())
	}
	_, on := browse(t, f.s, "tools", f.root, true)
	if !on.has("a.flac") || !on.has("B.FLAC") || on.has("c.mp3") || on.has("d.txt") {
		t.Errorf("files=1 listed %v", on.names())
	}
}

func TestBrowseSortsDirsFirstThenCaseInsensitive(t *testing.T) {
	f := newGuardFixture(t)
	for _, d := range []string{"zeta", "Alpha", "beta"} {
		if err := os.Mkdir(filepath.Join(f.root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	touch(t, filepath.Join(f.root, "a.flac"))
	_, got := browse(t, f.s, "tools", f.root, true)
	want := []string{"Alpha", "beta", "zeta", "a.flac"}
	for i, n := range got.names() {
		if i >= len(want) || n != want[i] {
			t.Fatalf("order = %v, want %v", got.names(), want)
		}
	}
}

func TestBrowseTruncatesAt2000(t *testing.T) {
	f := newGuardFixture(t)
	for i := 0; i < 2005; i++ {
		if err := os.Mkdir(filepath.Join(f.root, fmt.Sprintf("d%04d", i)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	_, got := browse(t, f.s, "tools", f.root, false)
	if len(got.Entries) != 2000 || !got.Truncated {
		t.Errorf("entries=%d truncated=%v", len(got.Entries), got.Truncated)
	}
}

func TestBrowseRejectsMissingOrUnknownScope(t *testing.T) {
	f := newGuardFixture(t)
	for _, scope := range []string{"", "all", "TOOLS"} {
		if code, _ := browse(t, f.s, scope, f.root, false); code != 400 {
			t.Errorf("scope %q: status %d, want 400", scope, code)
		}
	}
}

func TestBrowseFoldersListsOutsideRootsDirsOnly(t *testing.T) {
	f := newGuardFixture(t)
	other := t.TempDir()
	if err := os.Mkdir(filepath.Join(other, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	touch(t, filepath.Join(other, "song.flac"))
	touch(t, filepath.Join(other, "doc.txt"))
	if err := os.Symlink(filepath.Join(other, "sub"), filepath.Join(other, "linkdir")); err != nil {
		t.Fatal(err)
	}

	code, got := browse(t, f.s, "folders", other, true)
	if code != 200 {
		t.Fatalf("status %d", code)
	}
	if !got.has("sub") || !got.has("linkdir") || got.has("song.flac") || got.has("doc.txt") {
		t.Errorf("listed %v", got.names())
	}
	if got.Parent != filepath.Dir(other) {
		t.Errorf("parent = %q", got.Parent)
	}
}

func TestBrowseFoldersErrors(t *testing.T) {
	f := newGuardFixture(t)
	file := filepath.Join(t.TempDir(), "f.txt")
	touch(t, file)
	for name, c := range map[string]struct {
		path string
		want int
	}{
		"dotdot":   {f.root + "/../x", 403},
		"missing":  {filepath.Join(f.root, "nope"), 404},
		"file":     {file, 400},
		"relative": {"a/b", 400},
	} {
		if code, _ := browse(t, f.s, "folders", c.path, false); code != c.want {
			t.Errorf("%s: status %d, want %d", name, code, c.want)
		}
	}
}

func TestBrowseFoldersEmptyPathStartsAtHome(t *testing.T) {
	f := newGuardFixture(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	code, got := browse(t, f.s, "folders", "", false)
	if code != 200 || got.Path != home {
		t.Errorf("%d path=%q want %q", code, got.Path, home)
	}
}

func TestBrowseFoldersRootHasNoParent(t *testing.T) {
	f := newGuardFixture(t)
	code, got := browse(t, f.s, "folders", "/", false)
	if code != 200 || got.Parent != "" {
		t.Errorf("%d parent=%q", code, got.Parent)
	}
}

type flacResp struct {
	Paths     []string `json:"paths"`
	Truncated bool     `json:"truncated"`
}

func TestBrowseFlacRecursiveAndConfined(t *testing.T) {
	f := newGuardFixture(t)
	touch(t, filepath.Join(f.root, "a.flac"))
	touch(t, filepath.Join(f.root, "deep", "er", "b.flac"))
	touch(t, filepath.Join(f.root, "deep", "c.mp3"))

	var got flacResp
	resp := doRequest(t, f.s, "GET", "/api/browse/flac?path="+url.QueryEscape(f.root), nil, &got)
	if resp.StatusCode != 200 || len(got.Paths) != 2 {
		t.Fatalf("%d %+v", resp.StatusCode, got)
	}
	for _, p := range got.Paths {
		if p != filepath.Join(f.root, "a.flac") && p != filepath.Join(f.root, "deep", "er", "b.flac") {
			t.Errorf("unexpected path %q", p)
		}
	}
	for _, p := range f.badPaths() {
		resp := doRequest(t, f.s, "GET", "/api/browse/flac?path="+url.QueryEscape(p), nil, nil)
		if !isConfinementStatus(resp.StatusCode) {
			t.Errorf("%q: status %d", p, resp.StatusCode)
		}
	}
}

func TestBrowseFlacFiltersOutgoingFileSymlink(t *testing.T) {
	f := newGuardFixture(t)
	touch(t, filepath.Join(f.root, "ok.flac"))
	var got flacResp
	doRequest(t, f.s, "GET", "/api/browse/flac?path="+url.QueryEscape(f.root), nil, &got)
	for _, p := range got.Paths {
		if filepath.Base(p) == "link.flac" {
			t.Errorf("outgoing symlink listed: %v", got.Paths)
		}
	}
	if len(got.Paths) != 1 {
		t.Errorf("paths = %v", got.Paths)
	}
}

func TestBrowseRoutesRequireToken(t *testing.T) {
	s := newServerWith(t, func(c *ServerConfig) { c.AuthToken = "s3cret" })
	for _, target := range []string{"/api/browse?scope=folders&path=/tmp", "/api/browse/flac?path=/tmp"} {
		if resp := send(t, s, "GET", target, nil, ""); resp.StatusCode != 401 {
			t.Errorf("%s: status %d, want 401", target, resp.StatusCode)
		}
	}
}
