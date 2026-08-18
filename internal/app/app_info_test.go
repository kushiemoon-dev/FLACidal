package app

import "testing"

// Deliberately left uncovered here:
//   - CheckForUpdate: issues a live HTTP call to api.github.com and has no
//     injectable http.Client seam to intercept it.

func TestGetAppVersion(t *testing.T) {
	a := &App{version: "1.2.3"}
	if got := a.GetAppVersion(); got != "1.2.3" {
		t.Errorf("GetAppVersion() = %q, want %q", got, "1.2.3")
	}
}
