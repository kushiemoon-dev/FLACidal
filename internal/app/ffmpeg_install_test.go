package app

import (
	"errors"
	"sync"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func stubInstall(t *testing.T, fn func(ch chan<- core.FFmpegInstallProgress) error) {
	t.Helper()
	old := installFFmpeg
	installFFmpeg = fn
	t.Cleanup(func() { installFFmpeg = old })
}

func TestFFmpegInstallerRelaysProgressInOrder(t *testing.T) {
	stubInstall(t, func(ch chan<- core.FFmpegInstallProgress) error {
		defer close(ch)
		for i := 1; i <= 5; i++ {
			ch <- core.FFmpegInstallProgress{Percent: float64(i)}
		}
		return nil
	})
	var got []float64
	var i FFmpegInstaller
	if err := i.Install(func(p core.FFmpegInstallProgress) { got = append(got, p.Percent) }); err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("got %v, last event lost?", got)
	}
	for n, p := range got {
		if p != float64(n+1) {
			t.Fatalf("order: %v", got)
		}
	}
}

func TestFFmpegInstallerRefusesConcurrentInstall(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	stubInstall(t, func(ch chan<- core.FFmpegInstallProgress) error {
		defer close(ch)
		close(started)
		<-release
		return nil
	})
	var i FFmpegInstaller
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); _ = i.Install(func(core.FFmpegInstallProgress) {}) }()
	<-started
	if err := i.Install(func(core.FFmpegInstallProgress) {}); !errors.Is(err, ErrFFmpegInstallRunning) {
		t.Fatalf("err = %v", err)
	}
	close(release)
	wg.Wait()
}

func TestFFmpegInstallerRunningResetsAfterFailure(t *testing.T) {
	boom := errors.New("net down")
	stubInstall(t, func(ch chan<- core.FFmpegInstallProgress) error { close(ch); return boom })
	var i FFmpegInstaller
	if err := i.Install(func(core.FFmpegInstallProgress) {}); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if err := i.Install(func(core.FFmpegInstallProgress) {}); !errors.Is(err, boom) {
		t.Fatalf("second call should run again, got %v", err)
	}
}

func TestFFmpegInstallerResetsConverterOnlyOnSuccess(t *testing.T) {
	var resets int
	oldReset := resetConverter
	resetConverter = func() { resets++ }
	t.Cleanup(func() { resetConverter = oldReset })

	var i FFmpegInstaller
	stubInstall(t, func(ch chan<- core.FFmpegInstallProgress) error { close(ch); return errors.New("x") })
	_ = i.Install(func(core.FFmpegInstallProgress) {})
	if resets != 0 {
		t.Fatalf("reset after failure: %d", resets)
	}
	stubInstall(t, func(ch chan<- core.FFmpegInstallProgress) error { close(ch); return nil })
	_ = i.Install(func(core.FFmpegInstallProgress) {})
	if resets != 1 {
		t.Fatalf("resets = %d, want 1", resets)
	}
}
