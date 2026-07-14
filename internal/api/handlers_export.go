package api

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// handleExportFailedDownloads implements GET /api/downloads/export.
// Returns the current failed-download jobs as a downloadable CSV or TXT file
// (format=csv|txt, defaults to txt). Mirrors internal/app's
// App.ExportFailedDownloads, minus the native OS save dialog — the browser's
// own download flow (Content-Disposition: attachment) replaces it.
func (s *Server) handleExportFailedDownloads(c *fiber.Ctx) error {
	format := c.Query("format", "txt")
	if s.downloadManager == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "download manager not initialized"})
	}

	jobs := s.downloadManager.GetFailedJobs()
	if len(jobs) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no failed downloads to export"})
	}

	var sb strings.Builder
	filename := "failed_downloads.txt"
	contentType := "text/plain"

	if format == "csv" {
		filename = "failed_downloads.csv"
		contentType = "text/csv"
		sb.WriteString("artist,title,url,error\n")
		for _, job := range jobs {
			url := fmt.Sprintf("https://tidal.com/browse/track/%d", job.TrackID)
			sb.WriteString(fmt.Sprintf("%q,%q,%q,%q\n", job.Artist, job.Title, url, job.Error))
		}
	} else {
		for _, job := range jobs {
			url := fmt.Sprintf("https://tidal.com/browse/track/%d", job.TrackID)
			sb.WriteString(fmt.Sprintf("%s - %s | %s | %s\n", job.Artist, job.Title, url, job.Error))
		}
	}

	c.Set("Content-Type", contentType)
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	return c.SendString(sb.String())
}
