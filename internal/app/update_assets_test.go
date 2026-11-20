package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/datapointchris/goselfupdate"
)

// fakeSource serves one release and records which asset the updater downloads.
type fakeSource struct {
	assets     []string
	downloaded []string
}

func (f *fakeSource) LatestRelease(context.Context) (goselfupdate.Release, error) {
	rel := goselfupdate.Release{Tag: "v9.9.9"}
	for _, n := range f.assets {
		rel.Assets = append(rel.Assets, goselfupdate.Asset{Name: n, URL: "https://example.invalid/" + n})
	}
	return rel, nil
}

func (f *fakeSource) Download(_ context.Context, a goselfupdate.Asset) ([]byte, error) {
	f.downloaded = append(f.downloaded, a.Name)
	return []byte("#!/bin/sh\nexit 0\n"), nil
}

func updateWith(t *testing.T, assets []string) (*fakeSource, error) {
	t.Helper()
	src := &fakeSource{assets: assets}
	target := filepath.Join(t.TempDir(), "flacidal.AppImage")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := goselfupdate.UpdateTo(context.Background(), goselfupdate.Config{
		Owner: "o", Repo: "FLACidal", Binary: "FLACidal", Version: "v1.0.0",
		Source: src, Verifier: goselfupdate.NoVerification{},
	}, target)
	return src, err
}

var releaseAssets = []string{
	"flacidal_windows_amd64.exe",
	"flacidal.dmg",
	"flacidal_darwin_all.zip",
	"flacidal_linux_amd64.AppImage",
	"flacidal-x86_64.AppImage.zsync",
	"checksums.txt",
}

func TestUpdateAssetSelection_ZsyncDoesNotMakeLinuxAmbiguous(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("asset selection depends on the running platform")
	}
	src, err := updateWith(t, releaseAssets)
	if errors.Is(err, goselfupdate.ErrAmbiguousAsset) {
		t.Fatalf("the release is ambiguous for linux/amd64: %v", err)
	}
	// Installing the fake binary may fail after the download; only the choice matters here.
	if len(src.downloaded) != 1 || src.downloaded[0] != "flacidal_linux_amd64.AppImage" {
		t.Errorf("downloaded %v, want exactly flacidal_linux_amd64.AppImage", src.downloaded)
	}
}

// Control: it proves the test above can tell, and documents why the .zsync name must not contain "linux".
func TestUpdateAssetSelection_ZsyncNamedLinuxWouldBeAmbiguous(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("asset selection depends on the running platform")
	}
	assets := append(append([]string{}, releaseAssets[:4]...), "flacidal_linux_amd64.AppImage.zsync", "checksums.txt")
	if _, err := updateWith(t, assets); !errors.Is(err, goselfupdate.ErrAmbiguousAsset) {
		t.Errorf("err = %v, want ErrAmbiguousAsset", err)
	}
}
