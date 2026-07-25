package api

import (
	"github.com/gofiber/fiber/v2"

	"flacidal/internal/app"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// handleSetSourceOrder implements POST /api/sources/order.
// Mirrors internal/app's App.SetSourceOrder via the shared
// app.ValidateSourceOrder, except it has no equivalent of
// a.orchestrator.SetPriority — the Server struct has no orchestrator field
// (that's a Wails-app-only concern for live in-flight request routing), so
// only the persisted config + download manager priority are updated here.
func (s *Server) handleSetSourceOrder(c *fiber.Ctx) error {
	var req struct {
		Order []string `json:"order"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	validated, err := app.ValidateSourceOrder(req.Order)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if s.downloadManager != nil {
		s.downloadManager.SetSourceOrder(validated)
	}
	if s.config != nil {
		s.config.SourceOrder = validated
		if err := core.SaveConfig(s.config); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.JSON(fiber.Map{"success": true})
}
