package api

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"flacidal/internal/app"
)

func (s *Server) handleGetTrackHistory(c *fiber.Ctx) error {
	if s.db == nil {
		return c.JSON(fiber.Map{"entries": []interface{}{}, "total": 0})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	status := c.Query("status")
	if status != "" && status != "failed" && status != "completed" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid status"})
	}

	page, err := app.TrackHistoryPage(s.db, status, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(page)
}

func RegisterHistoryRoutes(api fiber.Router, s *Server) {
	api.Get("/track-history", s.handleGetTrackHistory)
}
