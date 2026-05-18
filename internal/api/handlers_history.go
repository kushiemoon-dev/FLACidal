package api

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// handleGetTrackHistory returns the per-track download log with pagination.
func (s *Server) handleGetTrackHistory(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	entries, err := s.db.ListHistory(limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	total, err := s.db.GetHistoryCount()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"entries": entries, "total": total})
}

// RegisterHistoryRoutes registers the per-track history route on the given router group.
func RegisterHistoryRoutes(api fiber.Router, s *Server) {
	api.Get("/track-history", s.handleGetTrackHistory)
}
