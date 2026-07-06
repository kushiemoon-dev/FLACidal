package main

import (
	"fmt"
	"path/filepath"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// =============================================================================
// Lyrics Methods (exposed to frontend)
// =============================================================================

// FetchLyrics fetches lyrics for a track from LRCLIB
func (a *App) FetchLyrics(title, artist string, durationSec int) (*core.Lyrics, error) {
	client := core.NewLyricsClient()
	lyrics, err := client.SearchLyrics(title, artist, durationSec)
	if err != nil {
		if a.logBuffer != nil {
			a.logBuffer.Warn(fmt.Sprintf("Lyrics not found for %s - %s", artist, title))
		}
		return nil, err
	}

	if a.logBuffer != nil {
		if lyrics.HasSynced {
			a.logBuffer.Success(fmt.Sprintf("Found synced lyrics for %s - %s", artist, title))
		} else {
			a.logBuffer.Success(fmt.Sprintf("Found plain lyrics for %s - %s", artist, title))
		}
	}

	return lyrics, nil
}

// FetchLyricsForFile fetches lyrics based on a FLAC file's metadata
func (a *App) FetchLyricsForFile(filePath string) (*core.Lyrics, error) {
	meta, err := core.ReadFLACMetadata(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	client := core.NewLyricsClient()
	return client.FetchLyricsForFile(meta)
}

// EmbedLyricsToFile embeds lyrics into a FLAC file
func (a *App) EmbedLyricsToFile(filePath string, plain, synced string) error {
	tagger := core.NewFLACTagger()
	err := tagger.EmbedLyrics(filePath, plain, synced)
	if err != nil {
		if a.logBuffer != nil {
			a.logBuffer.Error(fmt.Sprintf("Failed to embed lyrics: %s", err.Error()))
		}
		return err
	}

	if a.logBuffer != nil {
		a.logBuffer.Success(fmt.Sprintf("Lyrics embedded to %s", filepath.Base(filePath)))
	}
	return nil
}

// FetchAndEmbedLyrics fetches and embeds lyrics for a file in one operation
func (a *App) FetchAndEmbedLyrics(filePath string) (*core.Lyrics, error) {
	// Fetch lyrics based on file metadata
	lyrics, err := a.FetchLyricsForFile(filePath)
	if err != nil {
		return nil, err
	}

	// Embed lyrics
	err = a.EmbedLyricsToFile(filePath, lyrics.Plain, lyrics.Synced)
	if err != nil {
		return lyrics, err // Return lyrics even if embedding failed
	}

	// Save lyrics as separate file if enabled
	if a.config != nil && a.config.SaveLyricsFile {
		if saveErr := core.SaveLyricsFile(filePath, lyrics.Synced, lyrics.Plain); saveErr != nil {
			a.logBuffer.Warn("Failed to save lyrics file: " + saveErr.Error())
		}
	}

	return lyrics, nil
}

// FetchAndEmbedLyricsMultiple fetches and embeds lyrics for multiple files
func (a *App) FetchAndEmbedLyricsMultiple(filePaths []string) []map[string]interface{} {
	results := make([]map[string]interface{}, len(filePaths))

	for i, filePath := range filePaths {
		result := map[string]interface{}{
			"filePath": filePath,
			"success":  false,
		}

		lyrics, err := a.FetchAndEmbedLyrics(filePath)
		if err != nil {
			result["error"] = err.Error()
		} else {
			result["success"] = true
			result["hasPlain"] = lyrics.Plain != ""
			result["hasSynced"] = lyrics.HasSynced
		}

		results[i] = result
	}

	return results
}
