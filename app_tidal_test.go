package main

import (
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// Characterization tests for the "Tidal Methods" section of app.go.
//
// NOT tested here (documented, not fixed):
//   - FetchTidalPlaylist / RefreshTidalEndpoints: make live network calls to
//     the Tidal HiFi proxy / endpoint gist with no injectable HTTP seam.
//   - FetchTidalContent success branches (playlist/album/track/mix/artist):
//     each calls out to a.downloader/a.tidalClient over the network. Only the
//     invalid-URL error branch (which returns before touching those fields) is
//     exercised here.
//
// Bug note (not fixed): FetchTidalContent's `default:` case
// (`unsupported content type`) is unreachable — core.ParseTidalURL only ever
// returns "playlist", "track", "album", "artist", "mix", or a non-nil error,
// so contentType can never reach the switch's default arm.
//
// Bug note (not fixed): SetTidalCredentials dereferences a.config directly
// (`a.config.TidalClientID = clientID`) without the nil-guard used by sibling
// setters like SetDownloadFolder — calling it before a.config is initialized
// panics. Not reachable in production (startup() always sets a.config first),
// but inconsistent with the defensive pattern used elsewhere.

func TestSetTidalCredentials(t *testing.T) {
	core.SetDataDir(t.TempDir())
	a := &App{config: &core.Config{}}

	if err := a.SetTidalCredentials("client-id", "client-secret"); err != nil {
		t.Fatalf("SetTidalCredentials() error = %v", err)
	}
	if a.config.TidalClientID != "client-id" || a.config.TidalClientSecret != "client-secret" {
		t.Errorf("SetTidalCredentials() did not persist to config: %+v", a.config)
	}
	if a.tidalClient == nil {
		t.Error("SetTidalCredentials() did not initialize a.tidalClient")
	}
}

func TestSetTidalCredentials_NilConfigPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("SetTidalCredentials() with nil a.config: want panic (current behavior), got none")
		}
	}()
	a := &App{}
	_ = a.SetTidalCredentials("id", "secret")
}

func TestFetchTidalContent_InvalidURL(t *testing.T) {
	a := &App{}
	_, err := a.FetchTidalContent("https://not-a-tidal-url.example.com/whatever")
	if err == nil {
		t.Error("FetchTidalContent() with an invalid URL: want error, got nil")
	}
}

func TestValidateTidalURL(t *testing.T) {
	t.Run("invalid URL", func(t *testing.T) {
		a := &App{}
		got := a.ValidateTidalURL("not a url")
		if got["valid"] != false {
			t.Errorf("ValidateTidalURL(invalid) = %v, want valid=false", got)
		}
		if _, ok := got["error"]; !ok {
			t.Errorf("ValidateTidalURL(invalid) = %v, want an 'error' key", got)
		}
	})

	t.Run("valid playlist URL", func(t *testing.T) {
		a := &App{}
		got := a.ValidateTidalURL("https://tidal.com/browse/playlist/12345678-1234-1234-1234-123456789012")
		if got["valid"] != true {
			t.Errorf("ValidateTidalURL(valid playlist) = %v, want valid=true", got)
		}
		if got["type"] != "playlist" {
			t.Errorf("ValidateTidalURL(valid playlist) type = %v, want 'playlist'", got["type"])
		}
	})
}
