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
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/datapointchris/goselfupdate"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/mod/semver"
)

const forcedUpdateThreshold = 3

const (
	updateOwner = "kushiemoon-dev"
	updateRepo  = "flacidal"
)

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
func fetchReleaseTags(ctx context.Context, version, owner, repo string) (tags []string, releaseURL string, err error) {
	client := &http.Client{Timeout: 10 * time.Second}
	url := "https://api.github.com/repos/" + owner + "/" + repo + "/releases?per_page=100"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "FLACidal/"+version)

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

// buildUpdateStatus is the pure decision at the heart of GetUpdateStatus,
// split out so the forcedUpdateThreshold boundary is directly testable
// without a live network call.
func buildUpdateStatus(current string, tags []string, releaseURL string) *UpdateStatus {
	behind := countVersionsBehind(current, tags)
	return &UpdateStatus{
		HasUpdate:      behind > 0,
		CurrentVersion: current,
		LatestVersion:  strings.TrimPrefix(latestValidTag(tags), "v"),
		VersionsBehind: behind,
		Blocked:        behind >= forcedUpdateThreshold,
		ReleaseURL:     releaseURL,
	}
}

func (a *App) GetUpdateStatus() (*UpdateStatus, error) {
	tags, releaseURL, err := fetchReleaseTags(context.Background(), a.GetAppVersion(), updateOwner, updateRepo)
	if err != nil || len(tags) == 0 {
		return &UpdateStatus{CurrentVersion: a.GetAppVersion()}, nil // fail-open, see plan Conventions
	}
	return buildUpdateStatus(a.GetAppVersion(), tags, releaseURL), nil
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
		Owner:           updateOwner,
		Repo:            updateRepo,
		Binary:          updateRepo,
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

	if goruntime.GOOS == "darwin" {
		patchInfoPlistVersion(exe, strings.TrimPrefix(result.To, "v"))
	}

	if err := exec.Command(exe, os.Args[1:]...).Start(); err != nil {
		return err
	}
	// Quit (not os.Exit) so OnShutdown still runs: draining download
	// workers, saving config, closing the DB, same as any other exit path.
	wailsruntime.Quit(a.ctx)
	return nil
}

// patchInfoPlistVersion updates a macOS app bundle's Info.plist version keys
// to match the just-installed version. goselfupdate only replaces the raw
// Mach-O binary inside Contents/MacOS/, never Info.plist, so without this
// Finder's "Get Info" keeps showing the previous version even though the
// app itself (which reads its version from the embedded wails.json, not
// Info.plist) already reports correctly. Best-effort and cosmetic only: a
// failure here never blocks the update or relaunch.
func patchInfoPlistVersion(binaryPath, version string) {
	plistPath := filepath.Join(filepath.Dir(filepath.Dir(binaryPath)), "Info.plist")
	_ = exec.Command("plutil", "-replace", "CFBundleShortVersionString", "-string", version, plistPath).Run()
	_ = exec.Command("plutil", "-replace", "CFBundleVersion", "-string", version, plistPath).Run()
}
