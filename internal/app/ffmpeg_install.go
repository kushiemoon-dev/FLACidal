package app

import (
	"errors"
	"sync"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// ErrFFmpegInstallRunning is returned when an install is already in progress.
var ErrFFmpegInstallRunning = errors.New("an FFmpeg install is already running")

// Swapped in tests so no network is touched.
var (
	installFFmpeg  = core.InstallFFmpeg
	resetConverter = core.ResetConverter
)

// FFmpegInstaller runs at most one FFmpeg install at a time. Zero value is ready.
type FFmpegInstaller struct {
	mu      sync.Mutex
	running bool
}

// Install downloads FFmpeg, calling emit for each progress update in order.
// emit has returned for every update by the time Install does.
func (i *FFmpegInstaller) Install(emit func(core.FFmpegInstallProgress)) error {
	i.mu.Lock()
	if i.running {
		i.mu.Unlock()
		return ErrFFmpegInstallRunning
	}
	i.running = true
	i.mu.Unlock()
	defer func() {
		i.mu.Lock()
		i.running = false
		i.mu.Unlock()
	}()

	progressCh := make(chan core.FFmpegInstallProgress, 10)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for p := range progressCh {
			emit(p)
		}
	}()

	err := installFFmpeg(progressCh) // closes progressCh
	<-done
	if err != nil {
		return err
	}
	// So it picks up the newly installed binary's path.
	resetConverter()
	return nil
}

// SetFFmpegInstallForTest swaps the download and the converter reset so
// callers outside this package can test the install flow without network.
// The returned func restores the originals.
func SetFFmpegInstallForTest(install func(chan<- core.FFmpegInstallProgress) error, reset func()) (restore func()) {
	oldInstall, oldReset := installFFmpeg, resetConverter
	installFFmpeg, resetConverter = install, reset
	return func() { installFFmpeg, resetConverter = oldInstall, oldReset }
}
