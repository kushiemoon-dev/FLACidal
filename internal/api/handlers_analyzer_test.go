package api

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestSupportedAnalyzeExtensions(t *testing.T) {
	for _, ext := range []string{".flac", ".mp3", ".m4a", ".mp4", ".m4b", ".aac", ".wav", ".aiff", ".aif", ".ogg", ".opus", ".ape", ".wv", ".mpc"} {
		if !supportedAnalyzeExtensions[ext] {
			t.Errorf("expected %s to be a supported analyze extension", ext)
		}
	}
	if supportedAnalyzeExtensions[".exe"] {
		t.Error(".exe should not be a supported analyze extension")
	}
}

// TestHandleAnalyzeFile_JSONPath_MP3ReturnsNotApplicable reproduces the gap
// this closes: analyzing an MP3 by path (e.g. a Soulseek download) used to
// fail outright with "not a valid FLAC file" -- it should now succeed with a
// "not_applicable" verdict (no fake-lossless claim for a lossy format), same
// spectral/sample-rate data as any other analyzed file.
func TestHandleAnalyzeFile_JSONPath_MP3ReturnsNotApplicable(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not on PATH — skipping real-file analyzer test")
	}
	mp3Path := filepath.Join(t.TempDir(), "track.mp3")
	args := []string{"-f", "lavfi", "-i", "sine=frequency=440:duration=1:sample_rate=44100", "-y", mp3Path}
	if out, err := exec.Command(ffmpeg, args...).CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg MP3 fixture generation failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(mp3Path); err != nil {
		t.Fatalf("fixture missing: %v", err)
	}

	s := newTestServer(t)

	var result map[string]interface{}
	resp := doRequest(t, s, "POST", "/api/analyze", map[string]string{"path": mp3Path}, &result)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if result["verdict"] != "not_applicable" {
		t.Errorf("verdict = %v, want not_applicable", result["verdict"])
	}
}
