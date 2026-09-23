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

func TestSelectAssetForPlatform(t *testing.T) {
	assets := []releaseAsset{
		{Name: "flacidal.exe", BrowserDownloadURL: "https://example.com/flacidal.exe"},
		{Name: "flacidal.dmg", BrowserDownloadURL: "https://example.com/flacidal.dmg"},
		{Name: "flacidal.AppImage", BrowserDownloadURL: "https://example.com/flacidal.AppImage"},
	}

	tests := []struct {
		name string
		goos string
		want string
	}{
		{"windows", "windows", "https://example.com/flacidal.exe"},
		{"macos", "darwin", "https://example.com/flacidal.dmg"},
		{"linux", "linux", "https://example.com/flacidal.AppImage"},
		{"unknown platform", "plan9", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := selectAssetForPlatform(assets, tt.goos); got != tt.want {
				t.Errorf("selectAssetForPlatform(%q) = %q, want %q", tt.goos, got, tt.want)
			}
		})
	}
}

func TestSelectAssetForPlatform_NoMatchingAsset(t *testing.T) {
	assets := []releaseAsset{
		{Name: "flacidal.exe", BrowserDownloadURL: "https://example.com/flacidal.exe"},
	}
	if got := selectAssetForPlatform(assets, "darwin"); got != "" {
		t.Errorf("selectAssetForPlatform() = %q, want empty (no .dmg asset present)", got)
	}
}
