package app

import (
	"encoding/json"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

func (a *App) GetAppVersion() string {
	return a.version
}

type UpdateInfo struct {
	HasUpdate  bool   `json:"hasUpdate"`
	Version    string `json:"version"`
	URL        string `json:"url"`
	ReleaseURL string `json:"releaseUrl"`
}

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type release struct {
	TagName string         `json:"tag_name"`
	HTMLURL string         `json:"html_url"`
	Assets  []releaseAsset `json:"assets"`
}

// platformAssetExtension maps a runtime.GOOS value to the file extension
// used by this project's release assets. Returns "" for an unsupported OS.
func platformAssetExtension(goos string) string {
	switch goos {
	case "windows":
		return ".exe"
	case "darwin":
		return ".dmg"
	case "linux":
		return ".AppImage"
	default:
		return ""
	}
}

// selectAssetForPlatform returns the download URL of the first asset whose
// name matches the current platform's extension, or "" if none matches.
func selectAssetForPlatform(assets []releaseAsset, goos string) string {
	ext := platformAssetExtension(goos)
	if ext == "" {
		return ""
	}
	for _, a := range assets {
		if strings.HasSuffix(a.Name, ext) {
			return a.BrowserDownloadURL
		}
	}
	return ""
}

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

	var rel release

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &UpdateInfo{HasUpdate: false}, nil
	}

	if err := json.Unmarshal(body, &rel); err != nil {
		return &UpdateInfo{HasUpdate: false}, nil
	}

	latestVersion := strings.TrimPrefix(rel.TagName, "v")
	currentVersion := a.GetAppVersion()

	hasUpdate := semver.Compare("v"+latestVersion, "v"+currentVersion) > 0

	downloadURL := rel.HTMLURL
	if selected := selectAssetForPlatform(rel.Assets, runtime.GOOS); selected != "" {
		downloadURL = selected
	}

	return &UpdateInfo{
		HasUpdate:  hasUpdate,
		Version:    latestVersion,
		URL:        downloadURL,
		ReleaseURL: rel.HTMLURL,
	}, nil
}
