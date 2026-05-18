package api

// TODO: call RegisterHistoryRoutes from server.go's setupRoutes() to activate this handler.
// Example: RegisterHistoryRoutes(api, s)
// This registers GET /api/track-history as a per-track download log endpoint,
// distinct from the existing content-level /api/history routes in handlers.go.

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// handleGetTrackHistory returns the per-track download log with pagination.
func (s *Server) handleGetTrackHistory(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	if s.db == nil {
		return c.JSON(fiber.Map{"entries": []interface{}{}, "total": 0})
	}

	entries, err := s.db.ListHistory(limit, offset)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	total, err := s.db.GetHistoryCount()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"entries": entries,
		"total":   total,
	})
}

// RegisterHistoryRoutes registers the per-track history route on the given router group.
func RegisterHistoryRoutes(api fiber.Router, s *Server) {
	api.Get("/track-history", s.handleGetTrackHistory)
}
