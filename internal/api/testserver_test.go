package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// newTestServer builds a minimal Server wired with an in-memory config, ready
// to exercise handlers via s.app.Test(). Dependencies that individual tests
// don't need (DB, download manager, sources...) are left nil; handlers under
// test must not require them, or the test provides them explicitly.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	return NewServer(ServerConfig{
		Config:       &core.Config{},
		TidalSource:  core.NewTidalSource(),
		QobuzSource:  core.NewQobuzSource("", ""),
		LyricsClient: core.NewLyricsClient(),
	})
}

// doRequest performs an HTTP request against the test server and decodes the
// JSON response body into v (if v is non-nil). Returns the raw response.
func doRequest(t *testing.T, s *Server, method, path string, body interface{}, v interface{}) *http.Response {
	t.Helper()

	var reqBody *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(b)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			t.Fatalf("%s %s: decode response: %v", method, path, err)
		}
	}

	return resp
}
