package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/datapointchris/goselfupdate"
	"golang.org/x/mod/semver"
)

const forcedUpdateThreshold = 3

type UpdateStatus struct {
	HasUpdate      bool   `json:"hasUpdate"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	VersionsBehind int    `json:"versionsBehind"`
	Blocked        bool   `json:"blocked"`
	ReleaseURL     string `json:"releaseUrl"`
}

// countVersionsBehind returns how many tags in tagsNewestFirst are strictly
// newer than current by semver value, not by list position: GitHub orders
// releases by creation date, so a backported patch can appear before a
// later feature release. If current itself is not a valid semver value
// (e.g. "dev", an unreleased local build), it returns len(tagsNewestFirst)
// — trivially past forcedUpdateThreshold, since it can't be compared.
func countVersionsBehind(current string, tagsNewestFirst []string) int {
	cur := "v" + strings.TrimPrefix(current, "v")
	if !semver.IsValid(cur) {
		return len(tagsNewestFirst)
	}
	behind := 0
	for _, t := range tagsNewestFirst {
		tn := "v" + strings.TrimPrefix(t, "v")
		if semver.IsValid(tn) && semver.Compare(tn, cur) > 0 {
			behind++
		}
	}
	return behind
}

// latestValidTag returns the highest valid semver tag in tags, or "" if
// none is valid. Not necessarily tags[0]: see countVersionsBehind.
func latestValidTag(tags []string) string {
	best := ""
	for _, t := range tags {
		tn := "v" + strings.TrimPrefix(t, "v")
		if !semver.IsValid(tn) {
			continue
		}
		if best == "" || semver.Compare(tn, best) > 0 {
			best = tn
		}
	}
	return best
}

// fetchReleaseTags lists every release tag for owner/repo, newest first (the
// order GitHub already returns), plus the HTML URL of the newest release.
// Not tested directly: live network call, same convention as CheckForUpdate.
func fetchReleaseTags(ctx context.Context, owner, repo string) (tags []string, releaseURL string, err error) {
	client := &http.Client{Timeout: 10 * time.Second}
	url := "https://api.github.com/repos/" + owner + "/" + repo + "/releases?per_page=100"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("github releases API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	var releases []release
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, "", err
	}

	tags = make([]string, len(releases))
	for i, r := range releases {
		tags[i] = r.TagName
	}
	if len(releases) > 0 {
		releaseURL = releases[0].HTMLURL
	}
	return tags, releaseURL, nil
}

func (a *App) GetUpdateStatus() (*UpdateStatus, error) {
	tags, releaseURL, err := fetchReleaseTags(context.Background(), "kushiemoon-dev", "flacidal")
	if err != nil || len(tags) == 0 {
		return &UpdateStatus{CurrentVersion: a.GetAppVersion()}, nil // fail-open, see plan Conventions
	}
	behind := countVersionsBehind(a.GetAppVersion(), tags)
	return &UpdateStatus{
		HasUpdate:      behind > 0,
		CurrentVersion: a.GetAppVersion(),
		LatestVersion:  strings.TrimPrefix(latestValidTag(tags), "v"),
		VersionsBehind: behind,
		Blocked:        behind >= forcedUpdateThreshold,
		ReleaseURL:     releaseURL,
	}, nil
}

// appImagePath returns the running AppImage's own file path from the
// APPIMAGE environment variable AppImage itself sets at runtime, or "" when
// not running from one. goselfupdate's default target, os.Executable(),
// resolves inside an AppImage to the read-only squashfs mount it extracts
// itself into, not the AppImage file, and that mount cannot be replaced.
func appImagePath() string {
	return os.Getenv("APPIMAGE")
}

// DownloadAndInstallUpdate downloads, verifies and installs the latest
// release in place, then relaunches the app. goselfupdate handles asset
// selection (by GOOS/GOARCH), checksum verification and atomic binary
// replacement itself, including cleanup of any partial/corrupt download on
// failure — the caller only needs to surface the returned error.
// AllowPrerelease matches GetUpdateStatus counting every raw tag (plan
// Conventions): otherwise a prerelease-only "latest" would count as behind
// here but never be found by goselfupdate's own default (stable-only) view.
// Not tested directly: delegates to goselfupdate's own live network I/O and
// filesystem replacement, same convention as GetUpdateStatus/CheckForUpdate.
func (a *App) DownloadAndInstallUpdate() error {
	cfg := goselfupdate.Config{
		Owner:           "kushiemoon-dev",
		Repo:            "flacidal",
		Binary:          "flacidal",
		Version:         a.GetAppVersion(),
		AllowPrerelease: true,
	}

	target := appImagePath()
	var result goselfupdate.Result
	var err error
	if target != "" {
		result, err = goselfupdate.UpdateTo(context.Background(), cfg, target)
	} else {
		result, err = goselfupdate.Update(context.Background(), cfg)
	}
	if err != nil {
		return err
	}
	if !result.Applied {
		return errors.New("no update available: already running the latest version")
	}

	exe := target
	if exe == "" {
		exe, err = os.Executable()
		if err != nil {
			return err
		}
	}
	if err := exec.Command(exe, os.Args[1:]...).Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
