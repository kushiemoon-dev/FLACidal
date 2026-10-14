package api

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const secretContent = "do-not-touch"

type guardFixture struct {
	s       *Server
	root    string
	extra   string
	outside string // file outside every allowed root
}

func newGuardFixture(t *testing.T) guardFixture {
	t.Helper()
	root := t.TempDir()
	extra := t.TempDir()
	outDir := t.TempDir()
	outside := filepath.Join(outDir, "secret.flac")
	if err := os.WriteFile(outside, []byte(secretContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link.flac")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outDir, filepath.Join(root, "outdir")); err != nil {
		t.Fatal(err)
	}
	s := newTestServer(t)
	s.config.DownloadFolder = root
	s.config.ExternalLibraryPaths = []string{extra}
	return guardFixture{s: s, root: root, extra: extra, outside: outside}
}

func (f guardFixture) assertIntact(t *testing.T) {
	t.Helper()
	b, err := os.ReadFile(f.outside)
	if err != nil || string(b) != secretContent {
		t.Fatalf("outside file modified or removed: %q, %v", b, err)
	}
}

// badPaths are the four attack shapes every confined route must refuse.
func (f guardFixture) badPaths() map[string]string {
	return map[string]string{
		"dotdot":   f.root + "/../" + filepath.Base(filepath.Dir(f.outside)) + "/secret.flac",
		"symlink":  filepath.Join(f.root, "link.flac"),
		"absolute": f.outside,
		"relative": "secret.flac",
	}
}

func isConfinementStatus(code int) bool { return code == 400 || code == 403 }

func TestConfinedSinglePathRoutes(t *testing.T) {
	routes := []struct {
		name   string
		method string
		url    func(p string) string
		body   func(p string) interface{}
	}{
		{"delete", "DELETE", func(p string) string { return "/api/files?path=" + url.QueryEscape(p) }, nil},
		{"metadata", "GET", func(p string) string { return "/api/files/metadata?path=" + url.QueryEscape(p) }, nil},
		{"cover", "GET", func(p string) string { return "/api/files/cover?path=" + url.QueryEscape(p) }, nil},
		{"lyrics file", "POST", func(string) string { return "/api/lyrics/file" }, func(p string) interface{} { return map[string]string{"filePath": p} }},
		{"lyrics embed", "POST", func(string) string { return "/api/lyrics/embed" }, func(p string) interface{} { return map[string]string{"filePath": p, "plain": "x"} }},
		{"lyrics fetch-embed", "POST", func(string) string { return "/api/lyrics/fetch-embed" }, func(p string) interface{} { return map[string]string{"filePath": p} }},
		{"analyze", "POST", func(string) string { return "/api/analyze" }, func(p string) interface{} { return map[string]string{"path": p} }},
		{"analyze quick", "POST", func(string) string { return "/api/analyze/quick" }, func(p string) interface{} { return map[string]string{"path": p} }},
	}
	for _, r := range routes {
		f := newGuardFixture(t)
		for kind, p := range f.badPaths() {
			t.Run(r.name+"/"+kind, func(t *testing.T) {
				var body interface{}
				if r.body != nil {
					body = r.body(p)
				}
				resp := doRequest(t, f.s, r.method, r.url(p), body, nil)
				if !isConfinementStatus(resp.StatusCode) {
					t.Errorf("status = %d, want 400 or 403", resp.StatusCode)
				}
				f.assertIntact(t)
			})
		}
	}
}

func TestDeleteFile_OnlyFLAC(t *testing.T) {
	f := newGuardFixture(t)
	txt := filepath.Join(f.root, "notes.txt")
	os.WriteFile(txt, []byte("x"), 0o644)
	resp := doRequest(t, f.s, "DELETE", "/api/files?path="+url.QueryEscape(txt), nil, nil)
	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if _, err := os.Stat(txt); err != nil {
		t.Errorf("non-FLAC file was removed: %v", err)
	}
}

func TestConfinedRoutes_LegitimatePathsAccepted(t *testing.T) {
	f := newGuardFixture(t)
	for _, dir := range []string{f.root, f.extra} {
		fake := filepath.Join(dir, "ok.flac")
		os.WriteFile(fake, []byte("not flac"), 0o644)

		resp := doRequest(t, f.s, "GET", "/api/files/metadata?path="+url.QueryEscape(fake), nil, nil)
		if isConfinementStatus(resp.StatusCode) {
			t.Errorf("metadata under %s: status = %d, want past confinement", dir, resp.StatusCode)
		}
		resp = doRequest(t, f.s, "DELETE", "/api/files?path="+url.QueryEscape(fake), nil, nil)
		if resp.StatusCode != 200 {
			t.Errorf("delete under %s: status = %d, want 200", dir, resp.StatusCode)
		}
		if _, err := os.Stat(fake); !os.IsNotExist(err) {
			t.Errorf("file under %s not deleted", dir)
		}
	}
}

func TestConvert_OutputDirConfined(t *testing.T) {
	f := newGuardFixture(t)
	outDir := filepath.Dir(f.outside)
	for name, dir := range map[string]string{"outside": outDir, "relative": "out", "symlink": filepath.Join(f.root, "outdir")} {
		t.Run(name, func(t *testing.T) {
			resp := doRequest(t, f.s, "POST", "/api/convert", map[string]interface{}{
				"files": []string{filepath.Join(f.root, "a.flac")}, "format": "mp3", "outputDir": dir,
			}, nil)
			if !isConfinementStatus(resp.StatusCode) {
				t.Errorf("status = %d, want 400 or 403", resp.StatusCode)
			}
		})
	}
	// A missing folder under the root passes confinement (creating it is not this handler's concern).
	resp := doRequest(t, f.s, "POST", "/api/convert", map[string]interface{}{
		"files": []string{}, "format": "mp3", "outputDir": filepath.Join(f.root, "new"),
	}, nil)
	if isConfinementStatus(resp.StatusCode) && resp.StatusCode == 403 {
		t.Errorf("outputDir under root rejected with %d", resp.StatusCode)
	}
}

// fanOut covers routes where one bad path must not abort the batch: the bad
// entry carries the error, the valid one is processed, order is preserved.
func TestMultiFileRoutes_InvalidEntryIsolated(t *testing.T) {
	f := newGuardFixture(t)
	valid := filepath.Join(f.root, "ok.flac")
	os.WriteFile(valid, []byte("not flac"), 0o644)

	routes := []struct {
		name string
		url  string
		body map[string]interface{}
	}{
		{"rename preview", "/api/files/rename/preview", map[string]interface{}{"files": []string{f.outside, valid}, "template": "{title}"}},
		{"rename", "/api/files/rename", map[string]interface{}{"files": []string{f.outside, valid}, "template": "{title}"}},
		{"convert", "/api/convert", map[string]interface{}{"files": []string{f.outside, valid}, "format": "mp3"}},
		{"lyrics multiple", "/api/lyrics/fetch-embed/multiple", map[string]interface{}{"filePaths": []string{f.outside, valid}}},
		{"analyze multiple", "/api/analyze/multiple", map[string]interface{}{"paths": []string{f.outside, valid}}},
	}
	for _, r := range routes {
		t.Run(r.name, func(t *testing.T) {
			var results []map[string]interface{}
			resp := doRequest(t, f.s, "POST", r.url, r.body, &results)
			if resp.StatusCode != 200 {
				t.Fatalf("status = %d, want 200", resp.StatusCode)
			}
			if len(results) != 2 {
				t.Fatalf("got %d results, want 2", len(results))
			}
			if !strings.Contains(strings.ToLower(toJSON(results[0])), "outside the allowed folders") {
				t.Errorf("first entry lacks confinement error: %v", results[0])
			}
			f.assertIntact(t)
		})
	}
}

func TestRename_DestinationConfined(t *testing.T) {
	f := newGuardFixture(t)
	src := filepath.Join(f.root, "a.flac")
	writeTaggedFLAC(t, src, "Art", "Alb", "Tit")
	// root/outdir is a symlink to a folder outside the root.
	var results []map[string]interface{}
	doRequest(t, f.s, "POST", "/api/files/rename", map[string]interface{}{
		"files": []string{src}, "template": "outdir/{title}",
	}, &results)
	if len(results) != 1 || results[0]["success"] != false {
		t.Fatalf("results = %v, want one failure", results)
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("source moved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(f.outside), "Tit.flac")); !os.IsNotExist(err) {
		t.Errorf("file landed outside the root (stat err = %v)", err)
	}
}

func TestFolderAndConfigValidation(t *testing.T) {
	f := newGuardFixture(t)
	for _, folder := range []string{"relative", "/"} {
		resp := doRequest(t, f.s, "POST", "/api/folder", map[string]string{"folder": folder}, nil)
		if resp.StatusCode != 400 {
			t.Errorf("POST /api/folder %q: status = %d, want 400", folder, resp.StatusCode)
		}
	}
	for name, cfg := range map[string]map[string]interface{}{
		"relative download folder": {"downloadFolder": "rel"},
		"root download folder":     {"downloadFolder": "/"},
		"root library path":        {"externalLibraryPaths": []string{"/"}},
		"empty library path":       {"externalLibraryPaths": []string{""}},
	} {
		t.Run(name, func(t *testing.T) {
			resp := doRequest(t, f.s, "POST", "/api/config", cfg, nil)
			if resp.StatusCode != 400 {
				t.Errorf("status = %d, want 400", resp.StatusCode)
			}
		})
	}
	if f.s.config.DownloadFolder != f.root {
		t.Errorf("config changed by a rejected request: %q", f.s.config.DownloadFolder)
	}
}

func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// writeTaggedFLAC writes a minimal FLAC (magic + one VORBIS_COMMENT block).
func writeTaggedFLAC(t *testing.T, path, artist, album, title string) {
	t.Helper()
	le32 := func(n int) []byte { return []byte{byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24)} }
	comments := []string{"ARTIST=" + artist, "ALBUM=" + album, "TITLE=" + title}
	body := append(le32(0), le32(len(comments))...)
	for _, c := range comments {
		body = append(body, le32(len(c))...)
		body = append(body, c...)
	}
	data := append([]byte("fLaC"), 0x80|4, byte(len(body)>>16), byte(len(body)>>8), byte(len(body)))
	data = append(data, body...)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
