package api

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	core "github.com/kushiemoon-dev/flacidal-core"

	"flacidal/internal/app"
)

// probeRequestTimeout bounds one manual probe pass.
const probeRequestTimeout = 30 * time.Second

// healthSources is every source the headless server can report on. Amazon is
// not wired into the headless server yet.
func (s *Server) healthSources() app.HealthSources {
	return app.HealthSources{
		Downloader: s.downloader,
		Qobuz:      s.qobuzSource,
		Soulseek:   s.soulseekSource,
		Config:     s.config,
	}
}

func (s *Server) handleGetSourceHealth(c *fiber.Ctx) error {
	return c.JSON(sourceHealthOrEmpty(app.BuildSourceHealth(s.healthSources())))
}

func (s *Server) handleGetEndpointStatuses(c *fiber.Ctx) error {
	statuses := app.BuildEndpointStatuses(s.healthSources())
	if statuses == nil {
		statuses = []app.EndpointStatus{}
	}
	return c.JSON(statuses)
}

// handleProbeSources runs one probe pass over every endpoint, then returns the
// refreshed source health. Concurrent passes are serialized per pool.
func (s *Server) handleProbeSources(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), probeRequestTimeout)
	defer cancel()
	app.ProbeAllEndpoints(ctx, s.healthSources(), 0)
	return c.JSON(sourceHealthOrEmpty(app.BuildSourceHealth(s.healthSources())))
}

func sourceHealthOrEmpty(h []core.SourceHealth) []core.SourceHealth {
	if h == nil {
		return []core.SourceHealth{}
	}
	return h
}
