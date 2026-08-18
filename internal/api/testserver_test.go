package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Fields a given test has no use for (DB, download manager, sources, ...) are
// left as nil; a handler under test either has to tolerate that or the test
// supplies them.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	return NewServer(ServerConfig{
		Config:       &core.Config{},
		TidalSource:  core.NewTidalSource(),
		QobuzSource:  core.NewQobuzSource("", ""),
		LyricsClient: core.NewLyricsClient(),
	})
}

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
