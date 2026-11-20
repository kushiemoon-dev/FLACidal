package app

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// DiscographyDeps is what discography expansion and queuing need; Log may be nil.
type DiscographyDeps struct {
	Spotify    *core.SpotifyClient
	Tidal      *core.TidalSource
	Downloader *core.TidalHifiService
	Queue      *core.DownloadManager
	Log        *core.LogBuffer
}

var spotifyAlbumIDRe = regexp.MustCompile(`open\.spotify\.com/album/([^/?#]+)`)

func ExpandDiscography(d DiscographyDeps, rawURL string) ([]string, error) {
	info := core.ParseDiscographyURL(rawURL)
	if info == nil {
		return nil, fmt.Errorf("not a Spotify discography URL: %s", rawURL)
	}
	if d.Spotify == nil {
		return nil, fmt.Errorf("Spotify client not initialized")
	}
	urls, err := d.Spotify.FetchDiscographyAlbumURLs(info.ArtistID, info.Kind)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch discography for artist %s: %w", info.ArtistID, err)
	}
	if d.Log != nil {
		d.Log.Info(fmt.Sprintf("Discography expanded: %d albums for artist %s (kind=%s)", len(urls), info.ArtistID, info.Kind))
	}
	return urls, nil
}

func QueueDiscography(d DiscographyDeps, spotifyAlbumURLs []string, outputDir string) (int, error) {
	if d.Queue == nil {
		return 0, core.NewCodedError(core.CodeQueueNotReady, "")
	}
	if outputDir == "" {
		return 0, core.NewCodedError(core.CodeQueueNoOutputDir, "")
	}
	if d.Spotify == nil {
		return 0, fmt.Errorf("Spotify client not initialized")
	}
	if d.Tidal == nil {
		return 0, fmt.Errorf("Tidal source not initialized")
	}

	tidalClient := d.Tidal.GetAPIClient()

	queued := 0
	for _, albumURL := range spotifyAlbumURLs {
		m := spotifyAlbumIDRe.FindStringSubmatch(albumURL)
		if m == nil {
			continue
		}
		spotifyAlbumID := m[1]

		albumName, artistName, err := d.Spotify.GetAlbumMetadata(spotifyAlbumID)
		if err != nil {
			if d.Log != nil {
				d.Log.Warn(fmt.Sprintf("Discography: skipping %s, Spotify metadata lookup failed: %v", spotifyAlbumID, err))
			}
			continue
		}

		query := albumName + " " + artistName
		tidalAlbums, err := tidalClient.SearchAlbums(query, 5)
		if err != nil || len(tidalAlbums) == 0 {
			if d.Log != nil {
				d.Log.Warn(fmt.Sprintf("Discography: found no Tidal match for %q by %s", albumName, artistName))
			}
			continue
		}

		tidalAlbum := tidalAlbums[0]
		albumIDStr := strconv.Itoa(tidalAlbum.ID)

		album, err := d.Downloader.GetAlbumFromProxy(albumIDStr)
		if err != nil {
			if d.Log != nil {
				d.Log.Warn(fmt.Sprintf("Discography: unable to fetch Tidal album %s: %v", albumIDStr, err))
			}
			continue
		}

		artistFolder := core.SanitizeFileName(tidalAlbum.Artist)
		if artistFolder == "" {
			artistFolder = core.SanitizeFileName(artistName)
		}
		albumDir := filepath.Join(outputDir, artistFolder, core.SanitizeFileName(tidalAlbum.Title))
		if err := os.MkdirAll(albumDir, 0755); err != nil {
			continue
		}

		queued += d.Queue.QueueMultiple(album.Tracks, albumDir)
	}

	return queued, nil
}
