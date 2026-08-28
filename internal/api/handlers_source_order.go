package api

import (
	"github.com/gofiber/fiber/v2"

	"flacidal/internal/app"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Unlike internal/app's App.SetSourceOrder, this has nothing equivalent to
// a.orchestrator.SetPriority since Server carries no orchestrator field,
// that's specific to the Wails app's live in-flight request routing. Only
// the persisted config and the download manager's priority get updated here.
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
