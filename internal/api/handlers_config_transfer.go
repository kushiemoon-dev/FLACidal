package api

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"flacidal/internal/app"
)

// handleExportConfig sends the current config as a download, secrets blanked,
// in the same format as the desktop export.
func (s *Server) handleExportConfig(c *fiber.Ctx) error {
	data, err := app.BuildConfigExport(s.cfg())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not export config"})
	}
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="flacidal-config.json"`)
	return c.Send(data)
}

// handleImportConfig applies an export file. A secret left empty in the file
// keeps the stored one. Errors never echo the file content.
func (s *Server) handleImportConfig(c *fiber.Ctx) error {
	merged, err := app.MergeImportedConfig(c.Body(), s.cfg())
	if err != nil {
		if errors.Is(err, app.ErrNotConfigExport) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": app.ErrNotConfigExport.Error()})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid config file"})
	}
	merged, err = s.validateIncomingConfig(merged)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := s.applyConfig(merged); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not apply config"})
	}
	return c.JSON(fiber.Map{"success": true})
}
