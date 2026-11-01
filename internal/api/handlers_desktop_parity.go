package api

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	core "github.com/kushiemoon-dev/flacidal-core"

	"flacidal/internal/app"
)

func errorJSON(c *fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(fiber.Map{"error": msg})
}

func notReady(c *fiber.Ctx) error {
	return errorJSON(c, fiber.StatusServiceUnavailable, "service not ready")
}

// discographyDeps wires the shared helpers in internal/app to the Core components.
func (s *Server) discographyDeps() app.DiscographyDeps {
	return app.DiscographyDeps{
		Spotify:    s.core.SpotifySearch(),
		Tidal:      s.tidalSource,
		Downloader: s.downloader,
		Queue:      s.downloadManager,
		Log:        s.logBuffer,
	}
}

func (s *Server) handleDiscographyExpand(c *fiber.Ctx) error {
	var req struct {
		URL string `json:"url"`
	}
	if err := c.BodyParser(&req); err != nil {
		return errorJSON(c, fiber.StatusBadRequest, "invalid request body")
	}
	if core.ParseDiscographyURL(req.URL) == nil {
		return errorJSON(c, fiber.StatusBadRequest, "not a Spotify discography URL")
	}
	if s.core == nil {
		return notReady(c)
	}
	albums, err := app.ExpandDiscography(s.discographyDeps(), req.URL)
	if err != nil {
		return errorJSON(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"albums": albums})
}

func (s *Server) handleDiscographyQueue(c *fiber.Ctx) error {
	var req struct {
		AlbumURLs []string `json:"albumUrls"`
		OutputDir string   `json:"outputDir"`
	}
	if err := c.BodyParser(&req); err != nil {
		return errorJSON(c, fiber.StatusBadRequest, "invalid request body")
	}
	if req.OutputDir == "" {
		return errorJSON(c, fiber.StatusBadRequest, core.NewCodedError(core.CodeQueueNoOutputDir, "").Error())
	}
	if err := s.confine(req.OutputDir); err != nil {
		return pathError(c, err)
	}
	if s.core == nil {
		return notReady(c)
	}
	queued, err := app.QueueDiscography(s.discographyDeps(), req.AlbumURLs, req.OutputDir)
	if err != nil {
		return errorJSON(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"queued": queued})
}

func (s *Server) handleArtistAssets(c *fiber.Ctx) error {
	var req struct {
		ArtistID   string `json:"artistId"`
		ArtistName string `json:"artistName"`
		OutputDir  string `json:"outputDir"`
	}
	if err := c.BodyParser(&req); err != nil {
		return errorJSON(c, fiber.StatusBadRequest, "invalid request body")
	}
	if req.OutputDir == "" {
		return errorJSON(c, fiber.StatusBadRequest, core.NewCodedError(core.CodeQueueNoOutputDir, "").Error())
	}
	if err := s.confine(req.OutputDir); err != nil {
		return pathError(c, err)
	}
	if s.core == nil {
		return notReady(c)
	}
	count, err := app.DownloadArtistAssets(s.core.TidalClient(), req.ArtistID, req.ArtistName, req.OutputDir)
	if err != nil {
		return errorJSON(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"count": count})
}

// handleFFmpegInstall blocks until the install ends; progress goes out on /ws.
func (s *Server) handleFFmpegInstall(c *fiber.Ctx) error {
	err := s.ffmpeg.Install(func(p core.FFmpegInstallProgress) {
		s.wsHub.Broadcast(map[string]interface{}{
			"type": "ffmpeg-install-progress", "stage": p.Stage, "percent": p.Percent, "error": p.Error,
		})
	})
	switch {
	case errors.Is(err, app.ErrFFmpegInstallRunning):
		return errorJSON(c, fiber.StatusConflict, err.Error())
	case err != nil:
		return errorJSON(c, fiber.StatusInternalServerError, "FFmpeg install failed: "+err.Error())
	}
	return c.JSON(fiber.Map{"success": true})
}
