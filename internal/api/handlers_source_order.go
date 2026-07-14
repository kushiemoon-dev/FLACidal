package api

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// handleSetSourceOrder implements POST /api/sources/order.
// Mirrors internal/app's App.SetSourceOrder, except it has no equivalent of
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

	if len(req.Order) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "source order cannot be empty"})
	}
	validSources := map[string]bool{"tidal": true, "qobuz": true, "amazon": true, "bandcamp": true, "soulseek": true}
	seen := map[string]bool{}
	for _, src := range req.Order {
		if !validSources[src] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("unknown source: %s", src)})
		}
		if seen[src] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("duplicate source: %s", src)})
		}
		seen[src] = true
	}

	if s.downloadManager != nil {
		s.downloadManager.SetSourceOrder(req.Order)
	}
	if s.config != nil {
		s.config.SourceOrder = req.Order
		if err := core.SaveConfig(s.config); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.JSON(fiber.Map{"success": true})
}
