package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"sort"
	"strings"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func TestRunRegistersEverySource(t *testing.T) {
	h := startRun(t, nil)
	defer h.stop(t) //nolint:errcheck

	code, body := get(t, h.base+"/api/sources", "")
	if code != http.StatusOK {
		t.Fatalf("/api/sources = %d", code)
	}
	var infos []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(body), &infos); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, i := range infos {
		got = append(got, i.Name)
	}
	sort.Strings(got)
	want := []string{"amazon", "bandcamp", "deezer", "qobuz", "spotify", "tidal"}
	if !slices.Equal(got, want) {
		t.Errorf("sources = %v, want %v", got, want)
	}
}

func configOrder(t *testing.T, base string) []string {
	t.Helper()
	_, body := get(t, base+"/api/config", "")
	var cfg struct {
		SourceOrder []string `json:"sourceOrder"`
	}
	if err := json.Unmarshal([]byte(body), &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg.SourceOrder
}

func TestRunDefaultSourceOrderPutsSoulseekLast(t *testing.T) {
	h := startRun(t, nil)
	defer h.stop(t) //nolint:errcheck

	want := []string{"tidal", "qobuz", "amazon", "bandcamp", "soulseek"}
	if got := configOrder(t, h.base); !slices.Equal(got, want) {
		t.Errorf("default order = %v, want %v", got, want)
	}
}

func TestRunKeepsSoulseekFirstOrder(t *testing.T) {
	h := startRun(t, nil)
	defer h.stop(t) //nolint:errcheck

	resp, err := http.Post(h.base+"/api/sources/order", "application/json", strings.NewReader(`{"order":["soulseek","tidal"]}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/sources/order = %d", resp.StatusCode)
	}
	if got := configOrder(t, h.base); len(got) == 0 || got[0] != "soulseek" {
		t.Errorf("soulseek-first order not kept: %v", got)
	}
}

// The restored job is pointed at a closed local proxy so it fails at once
// instead of reaching the network.
func TestRunRestoresPersistedQueue(t *testing.T) {
	h := startRunWith(t, nil, func() {
		cfg := core.GetDefaultConfig()
		cfg.ProxyURL = "http://127.0.0.1:1"
		if err := core.SaveConfig(cfg); err != nil {
			t.Fatal(err)
		}
		db, err := core.NewDatabase()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		job := &core.DownloadJob{TrackID: 4242, Title: "Restored Title", Artist: "Someone"}
		if err := db.SaveQueueState([]*core.DownloadJob{job}, nil, nil); err != nil {
			t.Fatal(err)
		}
	})
	defer h.stop(t) //nolint:errcheck

	_, body := get(t, h.base+"/api/downloads/queue", "")
	var q struct{ Queued, Active, Failed int }
	if err := json.Unmarshal([]byte(body), &q); err != nil {
		t.Fatal(err)
	}
	if q.Queued+q.Active+q.Failed == 0 {
		t.Errorf("persisted job was not restored: %s", body)
	}
}

// Soulseek downloads use Core's native client, so enabling it registers the
// source even though no sldl binary exists.
func TestRunRegistersNativeSoulseekWhenEnabled(t *testing.T) {
	h := startRunWith(t, nil, func() {
		cfg := core.GetDefaultConfig()
		cfg.SoulseekEnabled = true
		if err := core.SaveConfig(cfg); err != nil {
			t.Fatal(err)
		}
	})
	defer h.stop(t) //nolint:errcheck

	_, body := get(t, h.base+"/api/sources", "")
	if !strings.Contains(body, `"name":"soulseek"`) {
		t.Errorf("soulseek not registered: %s", body)
	}
}
