package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// pausedManager returns a started but paused manager: queued jobs stay pending.
func pausedManager(t *testing.T) *core.DownloadManager {
	t.Helper()
	dm := core.NewDownloadManager(core.NewTidalHifiService(), 1)
	dm.Start()
	dm.PauseQueue()
	t.Cleanup(dm.Stop)
	return dm
}

func qualityServer(t *testing.T) (*Server, *core.DownloadManager) {
	t.Helper()
	dm := pausedManager(t)
	return NewServer(ServerConfig{
		Config:          &core.Config{},
		TidalSource:     core.NewTidalSource(),
		DownloadManager: dm,
	}), dm
}

func TestHandleQueueSingle_Quality(t *testing.T) {
	cases := []struct {
		name    string
		body    map[string]interface{}
		status  int
		quality string
		queued  int
	}{
		{"atmos", map[string]interface{}{"trackId": 1, "outputDir": "/tmp/x", "quality": "ATMOS"}, fiber.StatusOK, "ATMOS", 1},
		{"alias", map[string]interface{}{"trackId": 1, "outputDir": "/tmp/x", "quality": "dolby"}, fiber.StatusOK, "ATMOS", 1},
		{"absent", map[string]interface{}{"trackId": 1, "outputDir": "/tmp/x"}, fiber.StatusOK, "", 1},
		{"invalid", map[string]interface{}{"trackId": 1, "outputDir": "/tmp/x", "quality": "FLAC"}, fiber.StatusBadRequest, "", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, dm := qualityServer(t)
			var body map[string]interface{}
			resp := doRequest(t, s, "POST", "/api/downloads/single", tc.body, &body)
			if resp.StatusCode != tc.status {
				t.Fatalf("status = %d, want %d (%v)", resp.StatusCode, tc.status, body)
			}
			jobs := dm.GetPendingJobs()
			if len(jobs) != tc.queued {
				t.Fatalf("%d pending jobs, want %d", len(jobs), tc.queued)
			}
			if tc.queued == 1 && jobs[0].Quality != tc.quality {
				t.Errorf("Quality = %q, want %q", jobs[0].Quality, tc.quality)
			}
			if tc.status == fiber.StatusBadRequest && body["error"] != `unsupported download quality "FLAC"` {
				t.Errorf("error = %v", body["error"])
			}
		})
	}
}

func TestHandleQueueDownloads_Quality(t *testing.T) {
	s, dm := qualityServer(t)
	tracks := []core.TidalTrack{{ID: 1, Available: true}, {ID: 2, Available: true}}
	resp := doRequest(t, s, "POST", "/api/downloads/queue", map[string]interface{}{
		"tracks": tracks, "outputDir": "/tmp/x", "quality": "ATMOS",
	}, nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	jobs := dm.GetPendingJobs()
	if len(jobs) != 2 {
		t.Fatalf("%d pending jobs", len(jobs))
	}
	for _, j := range jobs {
		if j.Quality != "ATMOS" {
			t.Errorf("job %d Quality = %q", j.TrackID, j.Quality)
		}
	}

	resp = doRequest(t, s, "POST", "/api/downloads/queue", map[string]interface{}{
		"tracks": tracks, "outputDir": "/tmp/x", "quality": "FLAC",
	}, nil)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("invalid quality status = %d", resp.StatusCode)
	}
}

func TestHandleQueueArtistAlbum_InvalidQualityBeforeSideEffects(t *testing.T) {
	s, dm := qualityServer(t)
	out := filepath.Join(t.TempDir(), "out")
	resp := doRequest(t, s, "POST", "/api/downloads/queue/album", map[string]interface{}{
		"albumId": "1", "artistName": "A", "outputDir": out, "quality": "FLAC",
	}, nil)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if _, err := os.Stat(out); err == nil {
		if entries, _ := os.ReadDir(out); len(entries) > 0 {
			t.Errorf("album folder created before validation: %v", entries)
		}
	}
	if len(dm.GetPendingJobs()) != 0 {
		t.Error("nothing should be queued")
	}
}
