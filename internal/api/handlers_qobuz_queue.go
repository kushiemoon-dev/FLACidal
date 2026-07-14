package api

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// handleQueueQobuzDownloads implements POST /api/downloads/queue/qobuz.
// Mirrors internal/app's App.QueueQobuzDownloads.
func (s *Server) handleQueueQobuzDownloads(c *fiber.Ctx) error {
	var req struct {
		Tracks      []core.SourceTrack `json:"tracks"`
		OutputDir   string             `json:"outputDir"`
		ContentName string             `json:"contentName"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if s.downloadManager == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "download manager not initialized"})
	}
	if req.OutputDir == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no output directory specified"})
	}

	outputDir := req.OutputDir
	if req.ContentName != "" {
		outputDir = filepath.Join(outputDir, core.SanitizeFileName(req.ContentName))
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("failed to create folder: %v", err)})
		}
	}

	queued := s.downloadManager.QueueQobuzTracks(req.Tracks, outputDir)
	return c.JSON(fiber.Map{"queued": queued})
}
