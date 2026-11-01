package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func transferServer(t *testing.T, cfg core.Config, token string) *Server {
	t.Helper()
	core.SetDataDir(t.TempDir())
	return NewServer(ServerConfig{Config: &cfg, AuthToken: token})
}

func rawPost(t *testing.T, s *Server, path, body string, hdr map[string]string) (int, string) {
	t.Helper()
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := s.app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestConfigExportBlanksSecretsAndOmitsServerToken(t *testing.T) {
	s := transferServer(t, core.Config{
		TidalClientID: "tid", TidalClientSecret: "tsec", SoulseekUsername: "su", SoulseekPassword: "sp",
		QobuzAppID: "qid", QobuzAppSecret: "qsec", QobuzAuthToken: "qtok",
	}, "server-token-xyz")

	req := httptest.NewRequest("GET", "/api/config/export", nil)
	req.Header.Set("Authorization", "Bearer server-token-xyz")
	resp, err := s.app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); cd != `attachment; filename="flacidal-config.json"` {
		t.Errorf("Content-Disposition = %q", cd)
	}
	for _, secret := range []string{"tid", "tsec", "su", "sp", "qid", "qsec", "qtok", "server-token-xyz"} {
		if strings.Contains(string(body), `"`+secret+`"`) {
			t.Errorf("export leaks %s", secret)
		}
	}
	var out struct {
		SchemaVersion int         `json:"schemaVersion"`
		Config        core.Config `json:"config"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.SchemaVersion != 1 {
		t.Fatalf("bad export: %v %s", err, body)
	}
}

func TestConfigImportAppliesAndKeepsEmptySecrets(t *testing.T) {
	dl := t.TempDir()
	s := transferServer(t, core.Config{QobuzAppID: "kept-id", QobuzAppSecret: "kept-secret"}, "")

	file := `{"schemaVersion":1,"config":{"downloadFolder":"` + dl + `","language":"de","qobuzAppSecret":"new-secret"}}`
	code, body := rawPost(t, s, "/api/config/import", file, nil)
	if code != 200 {
		t.Fatalf("status %d: %s", code, body)
	}
	got := s.cfg()
	if got.Language != "de" || got.DownloadFolder != dl {
		t.Errorf("not applied: %+v", got)
	}
	if got.QobuzAppID != "kept-id" {
		t.Errorf("empty secret overwritten: %q", got.QobuzAppID)
	}
	if got.QobuzAppSecret != "new-secret" {
		t.Errorf("non-empty secret not replaced: %q", got.QobuzAppSecret)
	}
}

func TestConfigImportRejectsBadFilesWithoutEcho(t *testing.T) {
	s := transferServer(t, core.Config{}, "")
	for name, body := range map[string]string{
		"schema 2": `{"schemaVersion":2,"config":{"qobuzAppSecret":"LEAKME"}}`,
		"corrupt":  `{LEAKME not json`,
		"empty":    ``,
	} {
		code, resp := rawPost(t, s, "/api/config/import", body, nil)
		if code != 400 {
			t.Errorf("%s: status %d, want 400", name, code)
		}
		if strings.Contains(resp, "LEAKME") {
			t.Errorf("%s: response echoes content: %s", name, resp)
		}
	}
}

func TestConfigImportValidatesFolders(t *testing.T) {
	s := transferServer(t, core.Config{DownloadFolder: t.TempDir()}, "")
	for name, body := range map[string]string{
		"download root": `{"schemaVersion":1,"config":{"downloadFolder":"/"}}`,
		"library root":  `{"schemaVersion":1,"config":{"downloadFolder":"` + s.cfg().DownloadFolder + `","externalLibraryPaths":["/"]}}`,
	} {
		if code, resp := rawPost(t, s, "/api/config/import", body, nil); code != 400 {
			t.Errorf("%s: status %d, want 400 (%s)", name, code, resp)
		}
	}
}

func TestConfigTransferRoutesRequireToken(t *testing.T) {
	s := transferServer(t, core.Config{}, "s3cret")
	if resp := send(t, s, "GET", "/api/config/export", nil, ""); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("export: %d", resp.StatusCode)
	}
	if resp := send(t, s, "POST", "/api/config/import", nil, "{}"); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("import: %d", resp.StatusCode)
	}
}
