package api

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// extensionError maps a Core failure to a response: a coded error is the
// caller's to fix (400, "[code] detail" so the UI can translate it), anything
// else is a server fault.
func extensionError(c *fiber.Ctx, err error) error {
	var ce *core.CodedError
	status := fiber.StatusInternalServerError
	if errors.As(err, &ce) {
		status = fiber.StatusBadRequest
	}
	return c.Status(status).JSON(fiber.Map{"error": err.Error()})
}

// withExtensionCore wraps a handler so it answers 503 when there is no Core.
func (s *Server) withExtensionCore(h func(c *fiber.Ctx, k *core.Core) error) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if s.core == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "core unavailable"})
		}
		return h(c, s.core)
	}
}

func badBody(c *fiber.Ctx, err error) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
}

func (s *Server) registerExtensionRoutes(api fiber.Router) {
	api.Get("/extensions", s.withExtensionCore(func(c *fiber.Ctx, k *core.Core) error {
		return c.JSON(k.ListExtensions())
	}))
	api.Post("/extensions/install", s.withExtensionCore(func(c *fiber.Ctx, k *core.Core) error {
		var req struct {
			URL string `json:"url"`
		}
		if err := c.BodyParser(&req); err != nil {
			return badBody(c, err)
		}
		manifest, err := k.InstallExtension(req.URL)
		if err != nil {
			return extensionError(c, err)
		}
		return c.JSON(manifest)
	}))
	api.Post("/extensions/install/registry", s.withExtensionCore(func(c *fiber.Ctx, k *core.Core) error {
		var req struct {
			Repo string `json:"repo"`
			ID   string `json:"id"`
		}
		if err := c.BodyParser(&req); err != nil {
			return badBody(c, err)
		}
		manifest, err := k.InstallFromRegistry(req.Repo, req.ID)
		if err != nil {
			return extensionError(c, err)
		}
		return c.JSON(manifest)
	}))
	api.Get("/extensions/registries", s.withExtensionCore(func(c *fiber.Ctx, k *core.Core) error {
		return c.JSON(k.RegistrySources())
	}))
	api.Post("/extensions/registries", s.withExtensionCore(func(c *fiber.Ctx, k *core.Core) error {
		var req struct {
			URL string `json:"url"`
		}
		if err := c.BodyParser(&req); err != nil {
			return badBody(c, err)
		}
		if err := k.AddRegistrySource(req.URL); err != nil {
			return extensionError(c, err)
		}
		return c.JSON(fiber.Map{"success": true})
	}))
	api.Post("/extensions/registries/remove", s.withExtensionCore(func(c *fiber.Ctx, k *core.Core) error {
		var req struct {
			URL string `json:"url"`
		}
		if err := c.BodyParser(&req); err != nil {
			return badBody(c, err)
		}
		if err := k.RemoveRegistrySource(req.URL); err != nil {
			return extensionError(c, err)
		}
		return c.JSON(fiber.Map{"success": true})
	}))
	api.Post("/extensions/registry", s.withExtensionCore(func(c *fiber.Ctx, k *core.Core) error {
		var req struct {
			Repo string `json:"repo"`
		}
		if err := c.BodyParser(&req); err != nil {
			return badBody(c, err)
		}
		entries, err := k.BrowseRegistry(req.Repo)
		if err != nil {
			return extensionError(c, err)
		}
		return c.JSON(entries)
	}))
	api.Post("/extensions/:id/uninstall", s.withExtensionCore(func(c *fiber.Ctx, k *core.Core) error {
		if err := k.UninstallExtension(c.Params("id")); err != nil {
			return extensionError(c, err)
		}
		return c.JSON(fiber.Map{"success": true})
	}))
	api.Post("/extensions/:id/enabled", s.withExtensionCore(func(c *fiber.Ctx, k *core.Core) error {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := c.BodyParser(&req); err != nil {
			return badBody(c, err)
		}
		if err := k.SetExtensionEnabled(c.Params("id"), req.Enabled); err != nil {
			return extensionError(c, err)
		}
		return c.JSON(fiber.Map{"success": true})
	}))
	api.Post("/extensions/:id/auth", s.withExtensionCore(func(c *fiber.Ctx, k *core.Core) error {
		var req struct {
			Data map[string]string `json:"data"`
		}
		if err := c.BodyParser(&req); err != nil {
			return badBody(c, err)
		}
		if err := k.SetExtensionAuth(c.Params("id"), req.Data); err != nil {
			return extensionError(c, err)
		}
		return c.JSON(fiber.Map{"success": true})
	}))
	api.Get("/sources/order", s.withExtensionCore(func(c *fiber.Ctx, k *core.Core) error {
		return c.JSON(k.EffectiveSourceOrder())
	}))
}
