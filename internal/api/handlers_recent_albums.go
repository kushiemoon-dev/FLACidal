package api

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"flacidal/internal/app"
)

// handleGetRecentAlbums implements GET /api/history/recent.
// Mirrors internal/app's App.GetRecentAlbums via the shared app.RecentAlbums.
func (s *Server) handleGetRecentAlbums(c *fiber.Ctx) error {
	limit, err := strconv.Atoi(c.Query("limit", "24"))
	if err != nil || limit <= 0 {
		limit = 24
	}

	if s.db == nil {
		return c.JSON([]map[string]interface{}{})
	}

	albums, err := app.RecentAlbums(s.db, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(albums)
}
