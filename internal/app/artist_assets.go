package app

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// DownloadArtistAssets writes {outputDir}/{artistName}/ profile.jpg,
// profile_hires.jpg and banner.jpg, returning how many were saved.
func DownloadArtistAssets(tc *core.TidalClient, artistID, artistName, outputDir string) (int, error) {
	if outputDir == "" {
		return 0, core.NewCodedError(core.CodeQueueNoOutputDir, "")
	}
	if tc == nil {
		return 0, fmt.Errorf("Tidal client not initialized")
	}

	name, pictureID, err := tc.GetArtistPictureID(artistID)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch artist info: %w", err)
	}
	if pictureID == "" {
		return 0, fmt.Errorf("artist has no picture available")
	}

	if artistName == "" {
		artistName = name
	}

	urls := core.ArtistImageURLs(pictureID)
	if len(urls) == 0 {
		return 0, fmt.Errorf("no image URLs generated")
	}

	destDir := filepath.Join(outputDir, core.SanitizeFileName(artistName))
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create artist folder: %w", err)
	}

	fileNames := map[string]string{
		"profile":       "profile.jpg",
		"profile_hires": "profile_hires.jpg",
		"banner":        "banner.jpg",
	}

	client := &http.Client{}
	downloaded := 0
	for label, imgURL := range urls {
		destPath := filepath.Join(destDir, fileNames[label])

		resp, err := client.Get(imgURL)
		if err != nil || resp.StatusCode != 200 {
			if resp != nil {
				resp.Body.Close()
			}
			continue // this size isn't available, move on
		}

		f, err := os.Create(destPath)
		if err != nil {
			resp.Body.Close()
			continue
		}
		_, copyErr := io.Copy(f, resp.Body)
		f.Close()
		resp.Body.Close()
		if copyErr == nil {
			downloaded++
		}
	}

	return downloaded, nil
}
