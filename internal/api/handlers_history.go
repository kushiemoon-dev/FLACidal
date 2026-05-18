package api

// TODO: call RegisterHistoryRoutes from server.go's setupRoutes() to activate this handler.
// Example: RegisterHistoryRoutes(api, s)
// This registers GET /api/track-history as a per-track download log endpoint,
// distinct from the existing content-level /api/history routes in handlers.go.

import (
	"github.com/gofiber/fiber/v2"
)

// handleGetTrackHistory returns the per-track download log with pagination.
// TODO: restore ListHistory/GetHistoryCount calls when flacidal-core >= v0.4.5.
func (s *Server) handleGetTrackHistory(c *fiber.Ctx) error {
	// Stub: history persistence requires flacidal-core >= v0.4.5.
	return c.JSON(fiber.Map{"entries": []interface{}{}, "total": 0})
}

// RegisterHistoryRoutes registers the per-track history route on the given router group.
func RegisterHistoryRoutes(api fiber.Router, s *Server) {
	api.Get("/track-history", s.handleGetTrackHistory)
}
