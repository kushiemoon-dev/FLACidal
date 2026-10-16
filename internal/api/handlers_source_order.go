package api

import (
	"github.com/gofiber/fiber/v2"

	"flacidal/internal/app"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// With a Core the order reaches the config, the orchestrator and the download
// manager in one call. Without one (tests), only the persisted config and the
// download manager's priority are updated.
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

	if s.core != nil {
		if _, err := s.core.SetSourceOrder(validated); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true})
	}

	if s.downloadManager != nil {
		s.downloadManager.SetSourceOrder(validated)
	}
	if s.configLoaded() {
		next := s.updateConfig(func(c *core.Config) { c.SourceOrder = validated })
		if err := core.SaveConfig(&next); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.JSON(fiber.Map{"success": true})
}
