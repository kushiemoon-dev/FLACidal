package api

import (
	"github.com/gofiber/fiber/v2"

	"flacidal/internal/app"
)

// handleGetSldlStatus implements GET /api/sources/soulseek/status.
// Mirrors internal/app's App.GetSldlStatus via the shared app.SldlStatus.
func (s *Server) handleGetSldlStatus(c *fiber.Ctx) error {
	binaryPath := ""
	if s.config != nil {
		binaryPath = s.config.SoulseekBinaryPath
	}
	return c.JSON(app.SldlStatus(binaryPath))
}

// handleTestSoulseekConnection implements POST /api/sources/soulseek/test.
// Mirrors internal/app's App.TestSoulseekConnection via the shared
// app.TestSoulseekLogin. No server-side log buffer is wired up in headless
// mode (see GetLogs/ClearLogs known gap in lib/api.ts), so diagnostics are
// not surfaced anywhere here — only the JSON result matters.
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
