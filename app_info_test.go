package main

import "testing"

// Characterization test for the "App Info" section of app.go.
//
// NOT tested here (documented, not fixed):
//   - CheckForUpdate: makes a live HTTP call to api.github.com with no
//     injectable http.Client seam.

func TestGetAppVersion(t *testing.T) {
	a := &App{}
	if got := a.GetAppVersion(); got != appVersion {
		t.Errorf("GetAppVersion() = %q, want package var appVersion = %q", got, appVersion)
	}
}
