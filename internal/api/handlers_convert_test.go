package api

import (
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Tests for GET /api/convert/ffmpeg, GET /api/convert/available,
// GET /api/convert/formats and POST /api/convert.

func TestHandleIsConverterAvailable_MatchesCoreCheck(t *testing.T) {
	s := newTestServer(t)

	var body map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/convert/available", nil, &body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	want := core.IsConverterAvailable()
	if got, ok := body["available"].(bool); !ok || got != want {
		t.Errorf("available = %v, want %v (core.IsConverterAvailable())", body["available"], want)
	}
}

func TestHandleGetConversionFormats_NoConverterAvailable(t *testing.T) {
	if core.GetConverter() != nil {
		t.Skip("FFmpeg is available on this machine; the 'unavailable' branch isn't reachable here")
	}
	s := newTestServer(t)

	var formats []core.ConversionFormat
	resp := doRequest(t, s, "GET", "/api/convert/formats", nil, &formats)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if len(formats) != 0 {
		t.Errorf("formats = %v, want empty when no converter is available", formats)
	}
}

func TestHandleGetConversionFormats_ShapeMatchesCore(t *testing.T) {
	if core.GetConverter() == nil {
		t.Skip("FFmpeg not available on this machine; exercising the 'unavailable' branch instead")
	}
	s := newTestServer(t)

	var formats []core.ConversionFormat
	resp := doRequest(t, s, "GET", "/api/convert/formats", nil, &formats)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	want := core.GetConverter().GetFormats()
	if len(formats) != len(want) {
		t.Fatalf("formats = %d entries, want %d", len(formats), len(want))
	}
	for i, f := range formats {
		if f.ID != want[i].ID || len(f.Qualities) != len(want[i].Qualities) {
			t.Errorf("formats[%d] = %+v, want %+v", i, f, want[i])
		}
	}
}

func TestHandleGetFFmpegInfo(t *testing.T) {
	s := newTestServer(t)

	var body map[string]interface{}
	resp := doRequest(t, s, "GET", "/api/convert/ffmpeg", nil, &body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if body == nil {
		t.Error("body = nil, want a status map")
	}
}

func TestHandleConvertFiles_EmptyList(t *testing.T) {
	s := newTestServer(t)

	var results []core.ConversionResult
	resp := doRequest(t, s, "POST", "/api/convert", map[string]interface{}{
		"files":  []string{},
		"format": "mp3",
	}, &results)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if len(results) != 0 {
		t.Errorf("results = %v, want empty", results)
	}
}

func TestHandleConvertFiles_NonexistentSource(t *testing.T) {
	if core.GetConverter() == nil {
		t.Skip("FFmpeg not available on this machine; exercising the 'unavailable' branch instead")
	}
	s := newTestServer(t)

	var results []core.ConversionResult
	resp := doRequest(t, s, "POST", "/api/convert", map[string]interface{}{
		"files":     []string{"/tmp/does-not-exist-flacidal-test.flac"},
		"format":    "mp3",
		"quality":   "320k",
		"outputDir": "/tmp",
	}, &results)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if len(results) != 1 {
		t.Fatalf("results = %v, want 1 entry", results)
	}
	if results[0].Success {
		t.Errorf("results[0].Success = true, want false for a nonexistent source file")
	}
}

func TestHandleConvertFiles_NoConverterAvailable(t *testing.T) {
	if core.GetConverter() != nil {
		t.Skip("FFmpeg is available on this machine; the 'unavailable' branch isn't reachable here")
	}
	s := newTestServer(t)

	var results []core.ConversionResult
	resp := doRequest(t, s, "POST", "/api/convert", map[string]interface{}{
		"files":  []string{"/tmp/a.flac", "/tmp/b.flac"},
		"format": "mp3",
	}, &results)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if len(results) != 2 {
		t.Fatalf("results = %v, want 2 entries", results)
	}
	for _, r := range results {
		if r.Error != "FFmpeg not available" {
			t.Errorf("result.Error = %q, want %q", r.Error, "FFmpeg not available")
		}
	}
}
