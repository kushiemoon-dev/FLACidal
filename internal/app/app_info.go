package app

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// =============================================================================
// App Info
// =============================================================================

// GetAppVersion returns application version
func (a *App) GetAppVersion() string {
	return a.version
}

// UpdateInfo represents available update information
type UpdateInfo struct {
	HasUpdate  bool   `json:"hasUpdate"`
	Version    string `json:"version"`
	URL        string `json:"url"`
	ReleaseURL string `json:"releaseUrl"`
}

// CheckForUpdate checks GitHub for a newer release
func (a *App) CheckForUpdate() (*UpdateInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://api.github.com/repos/kushiemoon-dev/flacidal/releases/latest", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "FLACidal/"+a.GetAppVersion())

	resp, err := client.Do(req)
	if err != nil {
		return &UpdateInfo{HasUpdate: false}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return &UpdateInfo{HasUpdate: false}, nil
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &UpdateInfo{HasUpdate: false}, nil
	}

	if err := json.Unmarshal(body, &release); err != nil {
		return &UpdateInfo{HasUpdate: false}, nil
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := a.GetAppVersion()

	hasUpdate := latestVersion != currentVersion && latestVersion > currentVersion

	downloadURL := release.HTMLURL
	if len(release.Assets) > 0 {
		downloadURL = release.Assets[0].BrowserDownloadURL
	}

	return &UpdateInfo{
		HasUpdate:  hasUpdate,
		Version:    latestVersion,
		URL:        downloadURL,
		ReleaseURL: release.HTMLURL,
	}, nil
}
