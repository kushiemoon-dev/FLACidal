package api

import (
	"github.com/gofiber/fiber/v2"

	"flacidal/internal/app"
)

// handleTestSoulseekConnection logs in once through Core's built-in client.
// The login runs on a throwaway session, so it works for credentials that are
// not saved yet and never touches the live session.
func (s *Server) handleTestSoulseekConnection(c *fiber.Ctx) error {
	if s.core == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "core not initialised"})
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	res, err := s.core.TestSoulseekLogin(c.Context(), req.Username, req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(app.SoulseekLoginResultMap(res))
}
