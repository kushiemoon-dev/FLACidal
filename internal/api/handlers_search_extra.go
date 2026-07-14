package api

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"flacidal/internal/app"
)

// handleSearchTidalAlbums implements GET /api/content/search/albums.
// Mirrors internal/app's App.SearchTidalAlbums.
func (s *Server) handleSearchTidalAlbums(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Query parameter 'q' is required"})
	}
	if s.tidalSource == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "tidal source not initialized"})
	}

	limit, err := strconv.Atoi(c.Query("limit", "20"))
	if err != nil || limit <= 0 {
		limit = 20
	}

	albums, err := s.tidalSource.SearchAlbums(query, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(albums)
}

// handleSearchTidalArtists implements GET /api/content/search/artists.
// Mirrors internal/app's App.SearchTidalArtists.
func (s *Server) handleSearchTidalArtists(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Query parameter 'q' is required"})
	}
	if s.tidalSource == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "tidal source not initialized"})
	}

	limit, err := strconv.Atoi(c.Query("limit", "20"))
	if err != nil || limit <= 0 {
		limit = 20
	}

	artists, err := s.tidalSource.SearchArtists(query, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(artists)
}

// handleSearchDeezer implements GET /api/content/search/deezer.
// Mirrors internal/app's App.SearchDeezer via the shared app.SearchDeezerTracks.
func (s *Server) handleSearchDeezer(c *fiber.Ctx) error {
	query := c.Query("q")

	tracks, err := app.SearchDeezerTracks(query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(tracks)
}
