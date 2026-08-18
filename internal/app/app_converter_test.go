package app

import (
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Behavioral snapshot tests covering the converter methods section of app.go
// (the FFmpeg/conversion methods only — see app_sources_test.go for
// GetSourceHealth/InstallSldl/GetSldlStatus/TestSoulseekConnection, which
// live under app.go's "Converter Methods" comment header despite actually
// being about source health/Soulseek rather than conversion).
//
// Deliberately left uncovered here:
//   - InstallFFmpeg: pulls a real FFmpeg binary over the network and reports
//     progress through runtime.EventsEmit, which needs an actual Wails
//     runtime context.
//   - SelectFolderForConversion: pops a real native folder picker via the
//     Wails runtime.
//   - ConvertFiles' "converter available" branch depends on whether the test
//     machine actually has FFmpeg installed, so it's exercised adaptively
//     below instead of being assumed one way or the other.

func TestIsConverterAvailable(t *testing.T) {
	a := &App{}
	// This only confirms it mirrors core.IsConverterAvailable() without
	// panicking; actual FFmpeg presence depends on the test machine.
	if got, want := a.IsConverterAvailable(), core.IsConverterAvailable(); got != want {
		t.Errorf("IsConverterAvailable() = %v, want %v (core.IsConverterAvailable())", got, want)
	}
}

func TestGetFFmpegInfo(t *testing.T) {
	a := &App{}
	got := a.GetFFmpegInfo()
	if got == nil {
		t.Error("GetFFmpegInfo() = nil, want a status map")
	}
}

func TestGetFFmpegInstallStatus(t *testing.T) {
	a := &App{}
	got := a.GetFFmpegInstallStatus()
	if _, ok := got["systemAvailable"]; !ok {
		t.Errorf("GetFFmpegInstallStatus() = %v, want a systemAvailable key", got)
	}
	if _, ok := got["localInstalled"]; !ok {
		t.Errorf("GetFFmpegInstallStatus() = %v, want a localInstalled key", got)
	}
}

func TestGetConversionFormats(t *testing.T) {
	a := &App{}
	got := a.GetConversionFormats() // should never panic, whether or not FFmpeg is available
	if core.GetConverter() == nil && len(got) != 0 {
		t.Errorf("GetConversionFormats() with no converter = %v, want empty", got)
	}
}

func TestConvertFiles_NoConverterAvailable(t *testing.T) {
	if core.GetConverter() != nil {
		t.Skip("this machine has FFmpeg installed, so the 'unavailable' branch can't be exercised")
	}
	a := &App{}
	got := a.ConvertFiles([]string{"/tmp/a.flac", "/tmp/b.flac"}, "mp3", "320k", "/tmp/out", false)
	if len(got) != 2 {
		t.Fatalf("ConvertFiles() returned %d results, want 2", len(got))
	}
	for _, r := range got {
		if r.Error != "FFmpeg not available" {
			t.Errorf("ConvertFiles() result.Error = %q, want %q", r.Error, "FFmpeg not available")
		}
	}
}

func TestConvertFolder_NoFlacFiles(t *testing.T) {
	a := &App{}
	got := a.ConvertFolder(t.TempDir(), "mp3", "320k", "/tmp/out", false)
	if got != nil {
		t.Errorf("ConvertFolder() on an empty folder = %v, want nil", got)
	}
}
