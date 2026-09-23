package app

import "testing"

// Deliberately left uncovered here (same convention as app_info_test.go):
//   - GetUpdateStatus / fetchReleaseTags: issue a live HTTP call to
//     api.github.com and have no injectable http.Client seam to intercept it.
//   - DownloadAndInstallUpdate: delegates entirely to goselfupdate, which
//     performs its own live network I/O and binary replacement.

func TestCountVersionsBehind_CurrentAtHead(t *testing.T) {
	tags := []string{"v4.10.0", "v4.9.0", "v4.8.0"}
	if got := countVersionsBehind("v4.10.0", tags); got != 0 {
		t.Errorf("countVersionsBehind() = %d, want 0", got)
	}
}

func TestCountVersionsBehind_KnownPosition(t *testing.T) {
	// The exact case that breaks lexicographic comparison: "4.10.0" > "4.9.0"
	// is false as strings ('1' < '9') but true in semver.
	tags := []string{"v4.10.0", "v4.9.0", "v4.8.0"}
	if got := countVersionsBehind("v4.9.0", tags); got != 1 {
		t.Errorf("countVersionsBehind() = %d, want 1", got)
	}
}

func TestCountVersionsBehind_CurrentAbsent(t *testing.T) {
	tags := []string{"v4.10.0", "v4.9.0", "v4.8.0"}
	if got := countVersionsBehind("dev", tags); got != len(tags) {
		t.Errorf("countVersionsBehind() = %d, want %d (absent from list)", got, len(tags))
	}
}

func TestCountVersionsBehind_IgnoresMalformedTag(t *testing.T) {
	tags := []string{"v4.11.0", "not-a-version", "v4.10.0", "v4.9.0"}
	if got := countVersionsBehind("v4.9.0", tags); got != 2 {
		t.Errorf("countVersionsBehind() = %d, want 2 (malformed tag skipped, not counted or crashed on)", got)
	}
}

func TestCountVersionsBehind_AcceptsVersionWithoutVPrefix(t *testing.T) {
	tags := []string{"v4.10.0", "v4.9.0"}
	if got := countVersionsBehind("4.9.0", tags); got != 1 {
		t.Errorf("countVersionsBehind() = %d, want 1", got)
	}
}

func TestCountVersionsBehind_CurrentNewerThanEveryTag(t *testing.T) {
	// A local build ahead of the last tag (version bumped, not yet
	// released) must never be counted as behind, let alone blocked.
	tags := []string{"v4.10.0", "v4.9.0"}
	if got := countVersionsBehind("v4.11.0", tags); got != 0 {
		t.Errorf("countVersionsBehind() = %d, want 0 (current is newer than every tag)", got)
	}
}

func TestCountVersionsBehind_BackportOutOfChronologicalOrder(t *testing.T) {
	// GitHub's releases list is ordered by creation date, not semver: a
	// backport patch published after a newer feature release would land
	// first in tagsNewestFirst. Counting must go by version value, not
	// position, so this must not be miscounted.
	tags := []string{"v4.9.1", "v4.10.0", "v4.9.0"} // v4.9.1 backport created after v4.10.0
	if got := countVersionsBehind("v4.9.0", tags); got != 2 {
		t.Errorf("countVersionsBehind() = %d, want 2 (v4.9.1 and v4.10.0 are both newer)", got)
	}
}

func TestLatestValidTag_PicksHighestSemverNotFirstInList(t *testing.T) {
	tags := []string{"v4.9.1", "v4.10.0", "v4.9.0"} // v4.9.1 listed first (newest created), but v4.10.0 is the highest version
	if got := latestValidTag(tags); got != "v4.10.0" {
		t.Errorf("latestValidTag() = %q, want %q", got, "v4.10.0")
	}
}

func TestLatestValidTag_SkipsMalformedTags(t *testing.T) {
	tags := []string{"not-a-version", "v4.9.0"}
	if got := latestValidTag(tags); got != "v4.9.0" {
		t.Errorf("latestValidTag() = %q, want %q", got, "v4.9.0")
	}
}

func TestLatestValidTag_EmptyWhenNoneValid(t *testing.T) {
	tags := []string{"not-a-version"}
	if got := latestValidTag(tags); got != "" {
		t.Errorf("latestValidTag() = %q, want empty", got)
	}
}

func TestAppImagePath_ReturnsEnvValueWhenSet(t *testing.T) {
	t.Setenv("APPIMAGE", "/tmp/mount/usr/bin/flacidal.AppImage")
	if got := appImagePath(); got != "/tmp/mount/usr/bin/flacidal.AppImage" {
		t.Errorf("appImagePath() = %q, want the APPIMAGE env value", got)
	}
}

func TestAppImagePath_EmptyOutsideAppImage(t *testing.T) {
	t.Setenv("APPIMAGE", "")
	if got := appImagePath(); got != "" {
		t.Errorf("appImagePath() = %q, want empty when APPIMAGE is unset", got)
	}
}
