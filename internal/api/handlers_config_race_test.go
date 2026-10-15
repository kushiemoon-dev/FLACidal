package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// POST /api/config replaces s.config while other handlers read it. Run with
// -race.
func TestConfigConcurrentSaveAndRead(t *testing.T) {
	s := newTestServer(t)
	body, err := json.Marshal(core.Config{DownloadFolder: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			req := httptest.NewRequest("POST", "/api/config", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if resp, err := s.app.Test(req, -1); err != nil || resp.StatusCode != 200 {
				t.Errorf("POST /api/config: resp=%v err=%v", resp, err)
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			for _, path := range []string{"/api/config", "/api/folder"} {
				if _, err := s.app.Test(httptest.NewRequest("GET", path, nil), -1); err != nil {
					t.Errorf("GET %s: %v", path, err)
					return
				}
			}
		}
	}()
	wg.Wait()
}
