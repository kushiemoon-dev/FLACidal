package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	core "github.com/kushiemoon-dev/flacidal-core"

	"github.com/gofiber/fiber/v2"
)

type analyzeRequest struct {
	Path string `json:"path"`
}

func (s *Server) handleAnalyzeFileImpl(c *fiber.Ctx) error {
	filePath, tempPath, err := resolveAnalyzePath(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if tempPath != "" {
		defer cleanupTemp(tempPath)
	}

	result, err := core.AnalyzeAudioFile(filePath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(buildAnalyzeResponse(result))
}

func (s *Server) handleAnalyzeMultipleImpl(c *fiber.Ctx) error {
	var req struct {
		Paths []string `json:"paths"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if len(req.Paths) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "at least one path is required in the paths array"})
	}

	results := core.AnalyzeMultiple(req.Paths)

	responses := make([]fiber.Map, 0, len(results))
	for _, r := range results {
		rCopy := r
		responses = append(responses, buildAnalyzeResponse(&rCopy))
	}
	return c.JSON(responses)
}

func (s *Server) handleQuickAnalyzeImpl(c *fiber.Ctx) error {
	filePath, tempPath, err := resolveAnalyzePath(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if tempPath != "" {
		defer cleanupTemp(tempPath)
	}

	result, err := core.QuickAnalyze(filePath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(buildAnalyzeResponse(result))
}

// Call RegisterAnalyzerRoutes from setupRoutes() in place of the 501 stubs:
//
//	RegisterAnalyzerRoutes(api, s)
func RegisterAnalyzerRoutes(router fiber.Router, s *Server) {
	router.Post("/analyze", s.handleAnalyzeFileImpl)
	router.Post("/analyze/multiple", s.handleAnalyzeMultipleImpl)
	router.Post("/analyze/quick", s.handleQuickAnalyzeImpl)
}

// core.AnalyzeAudioFile runs full fake-lossless detection for FLAC; every
// other extension here only gets an ffprobe/spectral read with no lossless verdict.
var supportedAnalyzeExtensions = map[string]bool{
	".flac": true, ".mp3": true, ".m4a": true, ".mp4": true, ".m4b": true,
	".aac": true, ".wav": true, ".aiff": true, ".aif": true, ".ogg": true,
	".opus": true, ".ape": true, ".wv": true, ".mpc": true,
}

// When a multipart upload was used, the caller is responsible for deleting
// the temp file that gets written to /tmp.
func resolveAnalyzePath(c *fiber.Ctx) (filePath, tempPath string, err error) {
	file, uploadErr := c.FormFile("file")
	if uploadErr == nil {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if !supportedAnalyzeExtensions[ext] {
			return "", "", fmt.Errorf("unsupported file type: %s", ext)
		}
		tmp := fmt.Sprintf("/tmp/flacidal-analyze-%s", file.Filename)
		if saveErr := c.SaveFile(file, tmp); saveErr != nil {
			return "", "", fmt.Errorf("could not save uploaded file: %w", saveErr)
		}
		return tmp, tmp, nil
	}

	var req analyzeRequest
	if parseErr := c.BodyParser(&req); parseErr != nil {
		return "", "", fmt.Errorf("send either a multipart 'file' field or a JSON {\"path\": \"...\"} body")
	}
	if req.Path == "" {
		return "", "", fmt.Errorf("path is required")
	}
	if !filepath.IsAbs(req.Path) {
		return "", "", fmt.Errorf("path must be absolute")
	}
	return req.Path, "", nil
}

func buildAnalyzeResponse(r *core.AnalysisResult) fiber.Map {
	msg := r.Details
	if msg == "" {
		if r.IsTrueLossless {
			msg = "Genuinely lossless"
		} else {
			msg = fmt.Sprintf("Detected as upscaled lossy, spectral cutoff at %d Hz", r.SpectrumCutoff)
		}
	}

	format := strings.ToUpper(strings.TrimPrefix(filepath.Ext(r.FileName), "."))
	if format == "" {
		format = "FLAC"
	}

	return fiber.Map{
		// "isUpscaled" is only true for an actual upscale-detection verdict; a
		// lossy format reports "not_applicable" instead, since it never makes a
		// fake-lossless claim to begin with, distinct from IsTrueLossless == false.
		"isUpscaled":     r.Verdict == "upscaled" || r.Verdict == "likely_upscaled",
		"spectralCutoff": r.SpectrumCutoff,
		"format":         format,
		"message":        msg,
		"confidence":     int(r.Confidence),
		"verdict":        r.Verdict,
		"verdictLabel":   r.VerdictLabel,
		"fileName":       r.FileName,
		"sampleRate":     r.SampleRate,
		"bitDepth":       r.BitDepth,
	}
}

func cleanupTemp(path string) {
	os.Remove(path) //nolint:errcheck, deleting the temp upload is best-effort only
}
