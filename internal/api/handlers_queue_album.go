package api

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// handleQueueArtistAlbum implements POST /api/downloads/queue/album.
// Fetches a Tidal album's tracks and queues them all for download under an
// {Artist}/{Album} folder structure. Mirrors internal/app's App.QueueArtistAlbum.
func (s *Server) handleQueueArtistAlbum(c *fiber.Ctx) error {
	var req struct {
		AlbumID    string `json:"albumId"`
		ArtistName string `json:"artistName"`
		OutputDir  string `json:"outputDir"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if req.OutputDir == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no output directory specified"})
	}
	if s.downloadManager == nil || s.tidalSource == nil || s.tidalSource.GetService() == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "downloader not initialized"})
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

	queued := s.downloadManager.QueueMultiple(album.Tracks, albumDir)
	return c.JSON(fiber.Map{"queued": queued})
}
