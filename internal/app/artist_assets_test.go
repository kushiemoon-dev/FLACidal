package app

import (
	"errors"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func TestDownloadArtistAssetsGuards(t *testing.T) {
	var ce *core.CodedError
	if _, err := DownloadArtistAssets(nil, "1", "x", ""); !errors.As(err, &ce) || ce.Code != core.CodeQueueNoOutputDir {
		t.Errorf("empty outputDir: %v", err)
	}
	if _, err := DownloadArtistAssets(nil, "1", "x", "/out"); err == nil || err.Error() != "Tidal client not initialized" {
		t.Errorf("nil client: %v", err)
	}
}
