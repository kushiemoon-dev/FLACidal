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
	tags := []string{"v4.10.0", "not-a-version", "v4.9.0"}
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
