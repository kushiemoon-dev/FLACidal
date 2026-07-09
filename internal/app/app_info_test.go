package app

import "testing"

// Characterization test for the "App Info" section of app.go.
//
// NOT tested here (documented, not fixed):
//   - CheckForUpdate: makes a live HTTP call to api.github.com with no
//     injectable http.Client seam.

func TestGetAppVersion(t *testing.T) {
	a := &App{version: "1.2.3"}
	if got := a.GetAppVersion(); got != "1.2.3" {
		t.Errorf("GetAppVersion() = %q, want %q", got, "1.2.3")
	}
}
