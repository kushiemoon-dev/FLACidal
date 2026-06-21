package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	core "github.com/kushiemoon-dev/flacidal-core"

	"github.com/gofiber/fiber/v2"
)

// analyzeRequest accepts either a JSON body {"path": "/abs/path.flac"}
// or a multipart file upload (field name: "file", saved to /tmp).
type analyzeRequest struct {
	Path string `json:"path"`
}

// handleAnalyzeFileImpl implements POST /api/analyze.
// Accepts {"path":"/abs/path.flac"} or multipart upload (field "file").
func (s *Server) handleAnalyzeFileImpl(c *fiber.Ctx) error {
	filePath, tempPath, err := resolveAnalyzePath(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if tempPath != "" {
		defer cleanupTemp(tempPath)
	}

	result, err := core.AnalyzeFLAC(filePath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(buildAnalyzeResponse(result))
}

// handleAnalyzeMultipleImpl implements POST /api/analyze/multiple.
// Accepts {"paths": ["/abs/path1.flac", "/abs/path2.flac"]}.
func (s *Server) handleAnalyzeMultipleImpl(c *fiber.Ctx) error {
	var req struct {
		Paths []string `json:"paths"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if len(req.Paths) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "paths array is required"})
	}

	results := core.AnalyzeMultiple(req.Paths)

	responses := make([]fiber.Map, 0, len(results))
	for _, r := range results {
		rCopy := r
		responses = append(responses, buildAnalyzeResponse(&rCopy))
	}
	return c.JSON(responses)
}

// handleQuickAnalyzeImpl implements POST /api/analyze/quick.
// Accepts {"path": "/abs/path.flac"}.
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

// RegisterAnalyzerRoutes wires the real analyzer handlers onto an existing
// Fiber router group. Call this from setupRoutes() instead of the 501 stubs:
//
//	RegisterAnalyzerRoutes(api, s)
func RegisterAnalyzerRoutes(router fiber.Router, s *Server) {
	router.Post("/analyze", s.handleAnalyzeFileImpl)
	router.Post("/analyze/multiple", s.handleAnalyzeMultipleImpl)
	router.Post("/analyze/quick", s.handleQuickAnalyzeImpl)
}

// --- helpers ----------------------------------------------------------------

// resolveAnalyzePath returns the absolute file path to analyse.
// For multipart uploads the file is written to /tmp; the caller must remove it.
func resolveAnalyzePath(c *fiber.Ctx) (filePath, tempPath string, err error) {
	// Try multipart first
	file, uploadErr := c.FormFile("file")
	if uploadErr == nil {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext != ".flac" {
			return "", "", fmt.Errorf("only FLAC files are supported, got %s", ext)
		}
		tmp := fmt.Sprintf("/tmp/flacidal-analyze-%s", file.Filename)
		if saveErr := c.SaveFile(file, tmp); saveErr != nil {
			return "", "", fmt.Errorf("failed to save uploaded file: %w", saveErr)
		}
		return tmp, tmp, nil
	}

	// Fall back to JSON body {"path": "..."}
	var req analyzeRequest
	if parseErr := c.BodyParser(&req); parseErr != nil {
		return "", "", fmt.Errorf("provide either a multipart 'file' field or JSON {\"path\": \"...\"}")
	}
	if req.Path == "" {
		return "", "", fmt.Errorf("path is required")
	}
	if !filepath.IsAbs(req.Path) {
		return "", "", fmt.Errorf("path must be absolute")
	}
	return req.Path, "", nil
}

// buildAnalyzeResponse converts core.AnalysisResult → AnalyzeResponse fiber.Map.
func buildAnalyzeResponse(r *core.AnalysisResult) fiber.Map {
	msg := r.Details
	if msg == "" {
		if r.IsTrueLossless {
			msg = "Authentique lossless"
		} else {
			msg = fmt.Sprintf("Upscaled lossy détecté — coupure spectrale: %d Hz", r.SpectrumCutoff)
		}
	}

	return fiber.Map{
		"isUpscaled":     !r.IsTrueLossless,
		"spectralCutoff": r.SpectrumCutoff,
		"format":         "FLAC",
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
	os.Remove(path) //nolint:errcheck — best-effort cleanup of temp uploaded file
}
