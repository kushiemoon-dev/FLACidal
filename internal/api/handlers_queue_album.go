package api

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func (s *Server) handleQueueArtistAlbum(c *fiber.Ctx) error {
	var req struct {
		AlbumID    string `json:"albumId"`
		ArtistName string `json:"artistName"`
		OutputDir  string `json:"outputDir"`
		Quality    string `json:"quality"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	quality, qErr := core.NormalizeDownloadQuality(req.Quality)
	if qErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": qErr.Error()})
	}
	if req.OutputDir == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": core.NewCodedError(core.CodeQueueNoOutputDir, "").Error()})
	}
	if s.downloadManager == nil || s.tidalSource == nil || s.tidalSource.GetService() == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": core.NewCodedError(core.CodeQueueNotReady, "").Error()})
	}

	album, err := s.tidalSource.GetService().GetAlbumFromProxy(req.AlbumID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("failed to fetch album: %v", err)})
	}

	artistFolder := core.SanitizeFileName(req.ArtistName)
	if artistFolder == "" {
		artistFolder = core.SanitizeFileName(album.Artist)
	}
	albumFolder := core.SanitizeFileName(album.Title)
	albumDir := filepath.Join(req.OutputDir, artistFolder, albumFolder)
	if err := os.MkdirAll(albumDir, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("failed to create album folder: %v", err)})
	}

	queued := s.downloadManager.QueueMultipleWithQuality(album.Tracks, albumDir, quality)
	return c.JSON(fiber.Map{"queued": queued})
}
