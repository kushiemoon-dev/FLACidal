package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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

// countVersionsBehind returns how many entries of tagsNewestFirst (newest
// first, as returned by the GitHub releases API) separate current from the
// newest tag. Malformed tags are skipped. If current is not found in the
// list (older than everything returned, or a non-semver value like "dev"),
// it returns len(tagsNewestFirst) — trivially past forcedUpdateThreshold.
func countVersionsBehind(current string, tagsNewestFirst []string) int {
	cur := "v" + strings.TrimPrefix(current, "v")
	for i, t := range tagsNewestFirst {
		tn := "v" + strings.TrimPrefix(t, "v")
		if !semver.IsValid(tn) {
			continue
		}
		if semver.Compare(tn, cur) == 0 {
			return i
		}
	}
	return len(tagsNewestFirst)
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
		LatestVersion:  strings.TrimPrefix(tags[0], "v"),
		VersionsBehind: behind,
		Blocked:        behind >= forcedUpdateThreshold,
		ReleaseURL:     releaseURL,
	}, nil
}
