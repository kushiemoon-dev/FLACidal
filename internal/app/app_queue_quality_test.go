package app

import (
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func pausedDM(t *testing.T) *core.DownloadManager {
	t.Helper()
	dm := core.NewDownloadManager(core.NewTidalHifiService(), 1)
	dm.Start()
	dm.PauseQueue()
	t.Cleanup(dm.Stop)
	return dm
}

func TestQueueSingleDownload_InvalidQualityQueuesNothing(t *testing.T) {
	dm := pausedDM(t)
	a := &App{downloadManager: dm, downloader: core.NewTidalHifiService()}
	if err := a.QueueSingleDownload(1, "/tmp/out", "t", "a", "FLAC"); err == nil {
		t.Fatal("want an error for quality FLAC")
	}
	if n := len(dm.GetPendingJobs()); n != 0 {
		t.Errorf("%d jobs queued, want 0", n)
	}
}

func TestQueueDownloads_QualityApplied(t *testing.T) {
	dm := pausedDM(t)
	a := &App{downloadManager: dm}
	n, err := a.QueueDownloads([]core.TidalTrack{{ID: 1, Available: true}}, "/tmp/out", "", "", "", "atmos")
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if jobs := dm.GetPendingJobs(); len(jobs) != 1 || jobs[0].Quality != "ATMOS" {
		t.Errorf("unexpected jobs: %+v", jobs)
	}
	if _, err := a.QueueDownloads(nil, "/tmp/out", "", "", "", "bogus"); err == nil {
		t.Error("want an error for bogus quality")
	}
}
