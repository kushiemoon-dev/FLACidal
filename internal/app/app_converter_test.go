package app

import (
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Characterization tests for the "Converter Methods" section of app.go
// (the FFmpeg/conversion methods only — see app_sources_test.go for
// GetSourceHealth/InstallSldl/GetSldlStatus/TestSoulseekConnection, which
// share app.go's "Converter Methods" comment header in the original file
// despite being about source health/Soulseek, not conversion).
//
// NOT tested here (documented, not fixed):
//   - InstallFFmpeg: downloads a real FFmpeg binary over the network and
//     emits progress via runtime.EventsEmit, which requires a real Wails
//     runtime context.
//   - SelectFolderForConversion: opens a real native folder picker dialog via
//     the Wails runtime.
//   - ConvertFiles' "converter available" branch depends on whether FFmpeg is
//     actually installed on the machine running the tests, so it is
//     exercised adaptively below rather than assumed either way.

func TestIsConverterAvailable(t *testing.T) {
	a := &App{}
	// Just verify it reflects core.IsConverterAvailable() without panicking;
	// whether FFmpeg is actually present depends on the machine running tests.
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
	got := a.GetConversionFormats() // must not panic regardless of FFmpeg availability
	if core.GetConverter() == nil && len(got) != 0 {
		t.Errorf("GetConversionFormats() with no converter = %v, want empty", got)
	}
}

func TestConvertFiles_NoConverterAvailable(t *testing.T) {
	if core.GetConverter() != nil {
		t.Skip("FFmpeg is available on this machine; the 'unavailable' branch isn't reachable here")
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
