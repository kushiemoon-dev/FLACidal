package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	core "github.com/kushiemoon-dev/flacidal-core"
)

func newServerWith(t *testing.T, mod func(*ServerConfig)) *Server {
	t.Helper()
	cfg := ServerConfig{
		Config:       &core.Config{},
		TidalSource:  core.NewTidalSource(),
		QobuzSource:  core.NewQobuzSource("", ""),
		LyricsClient: core.NewLyricsClient(),
	}
	mod(&cfg)
	return NewServer(cfg)
}

func send(t *testing.T, s *Server, method, target string, hdr map[string]string, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range hdr {
		if strings.EqualFold(k, "Host") {
			req.Host = v
			continue
		}
		req.Header.Set(k, v)
	}
	resp, err := s.app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, target, err)
	}
	return resp
}

var upgradeHeaders = map[string]string{
	"Connection":            "Upgrade",
	"Upgrade":               "websocket",
	"Sec-WebSocket-Version": "13",
	"Sec-WebSocket-Key":     "dGhlIHNhbXBsZSBub25jZQ==",
}

func TestAuth_WriteRoutesRequireToken(t *testing.T) {
	s := newServerWith(t, func(c *ServerConfig) { c.AuthToken = "s3cret" })
	routes := []struct{ method, path string }{
		{"DELETE", "/api/files?path=/x.flac"},
		{"POST", "/api/files/rename"},
		{"POST", "/api/convert"},
		{"POST", "/api/folder"},
		{"POST", "/api/config"},
		{"POST", "/api/lyrics/file"},
		{"POST", "/api/lyrics/embed"},
		{"POST", "/api/lyrics/fetch-embed"},
		{"POST", "/api/lyrics/fetch-embed/multiple"},
		{"POST", "/api/analyze"},
		{"POST", "/api/analyze/quick"},
		{"POST", "/api/analyze/multiple"},
		{"POST", "/api/downloads/single"},
		{"POST", "/api/downloads/queue"},
		{"POST", "/api/history/clear"},
		{"DELETE", "/api/history/1"},
	}
	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			if got := send(t, s, r.method, r.path, nil, "{}").StatusCode; got != 401 {
				t.Errorf("no token: status = %d, want 401", got)
			}
			if got := send(t, s, r.method, r.path, map[string]string{"Authorization": "Bearer wrong"}, "{}").StatusCode; got != 401 {
				t.Errorf("wrong token: status = %d, want 401", got)
			}
		})
	}
}

func TestAuth_SensitiveReadsAndPublicRoutes(t *testing.T) {
	s := newServerWith(t, func(c *ServerConfig) { c.AuthToken = "s3cret" })
	good := map[string]string{"Authorization": "Bearer s3cret"}

	for _, p := range []string{"/api/config", "/api/files", "/api/files/metadata?path=/x.flac"} {
		if got := send(t, s, "GET", p, nil, "").StatusCode; got != 401 {
			t.Errorf("GET %s without token = %d, want 401", p, got)
		}
		if got := send(t, s, "GET", p, good, "").StatusCode; got == 401 {
			t.Errorf("GET %s with token = 401", p)
		}
	}
	if got := send(t, s, "GET", "/api/health", nil, "").StatusCode; got != 200 {
		t.Errorf("/api/health = %d, want 200", got)
	}
	if got := send(t, s, "GET", "/index.html", nil, "").StatusCode; got == 401 {
		t.Errorf("SPA path = 401, want public")
	}
	// The query token is only for WebSocket upgrades, never plain HTTP.
	if got := send(t, s, "GET", "/api/config?token=s3cret", nil, "").StatusCode; got != 401 {
		t.Errorf("query token on HTTP route = %d, want 401", got)
	}
}

func TestAuth_WebSocketRoutes(t *testing.T) {
	s := newServerWith(t, func(c *ServerConfig) { c.AuthToken = "s3cret" })
	for _, p := range []string{"/ws", "/ws/queue"} {
		t.Run(p, func(t *testing.T) {
			if got := send(t, s, "GET", p, upgradeHeaders, "").StatusCode; got != 401 {
				t.Errorf("upgrade without token = %d, want 401", got)
			}
			if got := send(t, s, "GET", p+"?token=wrong", upgradeHeaders, "").StatusCode; got != 401 {
				t.Errorf("upgrade with wrong token = %d, want 401", got)
			}
			if got := send(t, s, "GET", p, map[string]string{"Authorization": "Bearer s3cret"}, "").StatusCode; got != 426 {
				t.Errorf("authenticated non-upgrade = %d, want 426", got)
			}
		})
	}
}

// A real upgrade through app.Test starts the WebSocket handler, which trips a
// race inside the websocket library's connection pool. The middleware is
// therefore checked in isolation for the accepted-token case.
func TestTokenAuth_QueryTokenOnUpgrade(t *testing.T) {
	app := fiber.New()
	app.Use(tokenAuth("s3cret"))
	app.Get("/ws", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	for query, want := range map[string]int{"?token=s3cret": 200, "?token=nope": 401, "": 401} {
		req := httptest.NewRequest("GET", "/ws"+query, nil)
		for k, v := range upgradeHeaders {
			req.Header.Set(k, v)
		}
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != want {
			t.Errorf("query %q = %d, want %d", query, resp.StatusCode, want)
		}
	}
}

func TestAuth_DisabledWithoutToken(t *testing.T) {
	s := newTestServer(t)
	if got := send(t, s, "GET", "/api/config", nil, "").StatusCode; got != 200 {
		t.Errorf("status = %d, want 200 with auth disabled", got)
	}
}

func TestAuth_TokenNotExposed(t *testing.T) {
	s := newServerWith(t, func(c *ServerConfig) { c.AuthToken = "s3cret" })
	resp := send(t, s, "GET", "/api/config", map[string]string{"Authorization": "Bearer s3cret"}, "")
	b, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(b), "s3cret") {
		t.Error("token leaked in GET /api/config")
	}
}

func TestAuth_LoggerDoesNotCaptureQueryToken(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = f // the Fiber logger binds os.Stdout when the server is created
	s := newServerWith(t, func(c *ServerConfig) { c.AuthToken = "s3cret" })
	// A wrong token keeps the request from reaching the WebSocket handler
	// (see TestTokenAuth_QueryTokenOnUpgrade) while exercising the logger.
	send(t, s, "GET", "/ws?token=leakmark", upgradeHeaders, "")
	os.Stdout = orig

	b, _ := os.ReadFile(f.Name())
	if strings.Contains(string(b), "leakmark") {
		t.Errorf("token present in log output: %s", b)
	}
	if !strings.Contains(string(b), "/ws") {
		t.Errorf("request not logged: %q", b)
	}
}

func TestHostCheck(t *testing.T) {
	s := newServerWith(t, func(c *ServerConfig) { c.EnforceLocalHost = true })
	for _, h := range []string{"evil.example", "evil.example:8080", "localhost.evil.example"} {
		if got := send(t, s, "GET", "/api/health", map[string]string{"Host": h}, "").StatusCode; got != 403 {
			t.Errorf("Host %q = %d, want 403", h, got)
		}
	}
	if got := send(t, s, "GET", "/ws", mergeHdr(upgradeHeaders, map[string]string{"Host": "evil.example"}), "").StatusCode; got != 403 {
		t.Errorf("upgrade with foreign Host = %d, want 403", got)
	}
	for _, h := range []string{"localhost:8080", "127.0.0.1:8080", "[::1]:8080", "localhost", "LOCALHOST:80", "[::1]"} {
		if got := send(t, s, "GET", "/api/health", map[string]string{"Host": h}, "").StatusCode; got == 403 {
			t.Errorf("Host %q = 403, want allowed", h)
		}
	}
	// Opt-in: without the flag the default test Host (example.com) passes.
	if got := send(t, newTestServer(t), "GET", "/api/health", nil, "").StatusCode; got != 200 {
		t.Errorf("default server rejected example.com: %d", got)
	}
}

func mergeHdr(a, b map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func TestCORS(t *testing.T) {
	preflight := func(s *Server, origin string) *http.Response {
		return send(t, s, "OPTIONS", "/api/config", map[string]string{
			"Origin": origin, "Access-Control-Request-Method": "GET",
		}, "")
	}

	def := newTestServer(t)
	if got := preflight(def, "https://evil.example").Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("default Allow-Origin = %q, want none", got)
	}

	s := newServerWith(t, func(c *ServerConfig) { c.CORSOrigins = []string{"https://ok.example"} })
	ok := preflight(s, "https://ok.example")
	if got := ok.Header.Get("Access-Control-Allow-Origin"); got != "https://ok.example" {
		t.Errorf("configured origin Allow-Origin = %q", got)
	}
	if h := ok.Header.Get("Access-Control-Allow-Headers"); !strings.Contains(h, "Authorization") {
		t.Errorf("Allow-Headers = %q, want Authorization", h)
	}
	if got := preflight(s, "https://evil.example").Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("foreign origin Allow-Origin = %q, want none", got)
	}
}

func TestAuth_PathCaseAndSlashVariants(t *testing.T) {
	s := newServerWith(t, func(c *ServerConfig) { c.AuthToken = "s3cret" })
	for _, p := range []string{"/API/config", "/Api/config", "/api/CONFIG/", "//api/config", "/api//config"} {
		if got := send(t, s, "GET", p, nil, "").StatusCode; got == 200 {
			t.Errorf("GET %s without token = 200, want it blocked", p)
		}
	}
	for _, p := range []string{"/WS", "/Ws/queue"} {
		if got := send(t, s, "GET", p, upgradeHeaders, "").StatusCode; got != 401 {
			t.Errorf("upgrade %s without token = %d, want 401", p, got)
		}
	}
}

func TestOriginCheck_NoTokenMode(t *testing.T) {
	s := newServerWith(t, func(c *ServerConfig) {
		c.EnforceLocalHost = true
		c.CORSOrigins = []string{"https://ok.example"}
	})
	hdr := func(origin string) map[string]string {
		return map[string]string{"Host": "127.0.0.1:8080", "Origin": origin}
	}
	for _, o := range []string{"https://evil.example", "null", "http://localhost:8080"} {
		if got := send(t, s, "POST", "/api/folder", hdr(o), "folder=%2Fx").StatusCode; got != 403 {
			t.Errorf("Origin %q = %d, want 403", o, got)
		}
	}
	if got := send(t, s, "GET", "/ws", mergeHdr(upgradeHeaders, hdr("https://evil.example")), "").StatusCode; got != 403 {
		t.Errorf("cross-origin upgrade = %d, want 403", got)
	}
	for _, o := range []string{"http://127.0.0.1:8080", "https://ok.example"} {
		if got := send(t, s, "GET", "/api/health", hdr(o), "").StatusCode; got != 200 {
			t.Errorf("Origin %q = %d, want 200", o, got)
		}
	}
	if got := send(t, s, "GET", "/api/health", map[string]string{"Host": "127.0.0.1:8080"}, "").StatusCode; got != 200 {
		t.Errorf("no Origin = %d, want 200", got)
	}
}
