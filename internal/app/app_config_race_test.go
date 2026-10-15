package app

import (
	"sync"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// SaveConfig replaces a.config while other goroutines (queue workers, RPC
// handlers) read it. Run with -race.
func TestConfigConcurrentSaveAndRead(t *testing.T) {
	a := newTestApp(t)
	dirA, dirB := t.TempDir(), t.TempDir()
	a.config = &core.Config{DownloadFolder: dirA}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			folder := dirA
			if i%2 == 0 {
				folder = dirB
			}
			_ = a.SaveConfig(core.Config{DownloadFolder: folder})
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			_ = a.GetConfig().DownloadFolder
			_ = a.GetDownloadFolder()
		}
	}()
	wg.Wait()
}
