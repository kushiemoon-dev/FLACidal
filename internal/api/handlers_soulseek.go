package api

import (
	"github.com/gofiber/fiber/v2"

	"flacidal/internal/app"
)

func (s *Server) handleGetSldlStatus(c *fiber.Ctx) error {
	binaryPath := ""
	if s.config != nil {
		binaryPath = s.config.SoulseekBinaryPath
	}
	return c.JSON(app.SldlStatus(binaryPath))
}

// Headless mode has no server-side log buffer wired up (a known gap tracked
// alongside GetLogs/ClearLogs in lib/api.ts), so no diagnostics get surfaced
// here, only the JSON result matters.
func (s *Server) handleTestSoulseekConnection(c *fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	binaryPath := ""
	if s.config != nil {
		binaryPath = s.config.SoulseekBinaryPath
	}
	return c.JSON(app.TestSoulseekLogin(binaryPath, req.Username, req.Password, nil))
}
