package api

import (
	"fmt"
	"path/filepath"

	"github.com/gofiber/fiber/v2"

	"flacidal/internal/app"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func tooManyFiles(c *fiber.Ctx, n int) error {
	return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
		"error": fmt.Sprintf("too many files: %d (maximum %d)", n, core.MaxAnalyzeBatch),
	})
}

func (s *Server) broadcastAnalyzeProgress(batchID string, done, total int, result core.AnalysisResult) {
	msg := map[string]interface{}{
		"type": "analyze-progress", "batchId": batchID, "done": done, "total": total, "result": result,
	}
	if s.analyzeProgressHook != nil {
		s.analyzeProgressHook(msg)
		return
	}
	s.wsHub.Broadcast(msg)
}

// handleAnalyzeBatch analyzes paths with bounded concurrency, streaming
// progress on /ws. Paths outside the allowed folders become error rows.
func (s *Server) handleAnalyzeBatch(c *fiber.Ctx) error {
	var req struct {
		BatchID string   `json:"batchId"`
		Paths   []string `json:"paths"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if req.BatchID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "a batchId is required"})
	}
	if len(req.Paths) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "at least one path is required in the paths array"})
	}
	if len(req.Paths) > core.MaxAnalyzeBatch {
		return tooManyFiles(c, len(req.Paths))
	}
	ctx, release, err := s.analyzeBatches.Begin(req.BatchID)
	if err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}
	defer release()

	total := len(req.Paths)
	results := make([]core.AnalysisResult, 0, total)
	var valid []string
	for _, p := range req.Paths {
		if err := s.confine(p); err != nil {
			row := core.AnalysisResult{FilePath: p, FileName: filepath.Base(p), Verdict: "error", VerdictLabel: "Error", Details: err.Error()}
			results = append(results, row)
			s.broadcastAnalyzeProgress(req.BatchID, len(results), total, row)
			continue
		}
		valid = append(valid, p)
	}
	refused := len(results)
	analyzed := core.AnalyzeBatch(ctx, valid, core.DefaultAnalyzeWorkers(), app.EnrichAnalysis, func(p core.BatchProgress) {
		s.broadcastAnalyzeProgress(req.BatchID, refused+p.Done, total, p.Result)
	})
	results = append(results, analyzed...)

	return c.JSON(fiber.Map{
		"results":   results,
		"cancelled": len(results) < total,
		"total":     total,
	})
}

func (s *Server) handleCancelAnalyzeBatch(c *fiber.Ctx) error {
	if !s.analyzeBatches.Cancel(c.Params("id")) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no such running batch"})
	}
	return c.JSON(fiber.Map{"cancelled": true})
}

// handleAnalyzeReport turns posted results into a CSV or JSON download. It
// reads nothing from disk, so no path confinement applies.
func (s *Server) handleAnalyzeReport(c *fiber.Ctx) error {
	var req struct {
		Results []core.AnalysisResult `json:"results"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if len(req.Results) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "results must not be empty"})
	}
	if len(req.Results) > core.MaxAnalyzeBatch {
		return tooManyFiles(c, len(req.Results))
	}
	data, contentType, ext, err := core.BuildAnalysisReport(req.Results, c.Query("format"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	c.Set(fiber.HeaderContentType, contentType)
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="analysis_report.%s"`, ext))
	return c.Send(data)
}
