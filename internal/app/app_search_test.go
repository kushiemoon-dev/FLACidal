package app

import "testing"

// Characterization tests for the "Search Methods" section of app.go.
//
// NOT tested here (documented, not fixed):
//   - SearchTidal / SearchTidalAlbums / SearchTidalArtists success paths, and
//     SearchDeezer's non-empty-query path: all make live network calls
//     (Tidal proxy, Deezer public API) with no injectable HTTP seam. Only
//     nil-guard / early-return branches are exercised.

func TestSearchTidal_NilDownloader(t *testing.T) {
	a := &App{}
	if _, err := a.SearchTidal("daft punk"); err == nil {
		t.Error("SearchTidal() with nil downloader: want error, got nil")
	}
}

func TestSearchTidalAlbums_NilTidalSource(t *testing.T) {
	a := &App{}
	if _, err := a.SearchTidalAlbums("daft punk"); err == nil {
		t.Error("SearchTidalAlbums() with nil tidalSource: want error, got nil")
	}
}

func TestSearchTidalArtists_NilTidalSource(t *testing.T) {
	a := &App{}
	if _, err := a.SearchTidalArtists("daft punk"); err == nil {
		t.Error("SearchTidalArtists() with nil tidalSource: want error, got nil")
	}
}

func TestSearchDeezer_EmptyQuery(t *testing.T) {
	a := &App{}
	got, err := a.SearchDeezer("")
	if err != nil {
		t.Fatalf("SearchDeezer(\"\") error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("SearchDeezer(\"\") = %v, want empty (no network call)", got)
	}
}
