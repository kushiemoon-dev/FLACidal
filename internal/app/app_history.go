package app

import (
	"fmt"
	"sort"
	"strings"
	"time"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// =============================================================================
// Database Methods (exposed to frontend)
// =============================================================================

// GetCacheStats returns track cache statistics
func (a *App) GetCacheStats() map[string]interface{} {
	if a.db == nil {
		return map[string]interface{}{"error": "database not initialized"}
	}

	total, byMethod, err := a.db.GetCacheStats()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"total":    total,
		"byMethod": byMethod,
	}
}

// GetDownloadHistory returns all download history
func (a *App) GetDownloadHistory() ([]core.DownloadRecord, error) {
	if a.db == nil {
		return nil, nil
	}
	return a.db.GetAllDownloadRecords()
}

// GetRecentAlbums returns deduplicated recent album downloads for the home page grid
func (a *App) GetRecentAlbums(limit int) ([]map[string]interface{}, error) {
	if a.db == nil {
		return []map[string]interface{}{}, nil
	}
	records, err := a.db.GetAllDownloadRecords()
	if err != nil {
		return nil, fmt.Errorf("GetRecentAlbums: %w", err)
	}

	// Sort by most recent first
	sort.Slice(records, func(i, j int) bool {
		return records[i].LastDownloadAt.After(records[j].LastDownloadAt)
	})

	result := make([]map[string]interface{}, 0, limit)
	seen := make(map[string]bool)
	for _, r := range records {
		if len(result) >= limit {
			break
		}
		key := r.TidalContentID
		if seen[key] {
			continue
		}
		seen[key] = true

		// Parse "Artist — Title" if present, else use full name as title
		title := r.TidalContentName
		artist := ""
		if parts := strings.SplitN(r.TidalContentName, " — ", 2); len(parts) == 2 {
			artist = parts[0]
			title = parts[1]
		}

		result = append(result, map[string]interface{}{
			"title":         title,
			"artist":        artist,
			"cover_url":     "",
			"track_count":   r.TracksTotal,
			"source":        r.ContentType,
			"content_id":    r.TidalContentID,
			"content_type":  r.ContentType,
			"downloaded_at": r.LastDownloadAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

// GetDownloadHistoryFiltered returns filtered download history with pagination
func (a *App) GetDownloadHistoryFiltered(filter map[string]interface{}) (map[string]interface{}, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Parse filter options
	dbFilter := core.HistoryFilter{}

	if ct, ok := filter["contentType"].(string); ok {
		dbFilter.ContentType = ct
	}
	if search, ok := filter["search"].(string); ok {
		dbFilter.Search = search
	}
	if limit, ok := filter["limit"].(float64); ok {
		dbFilter.Limit = int(limit)
	}
	if offset, ok := filter["offset"].(float64); ok {
		dbFilter.Offset = int(offset)
	}

	records, total, err := a.db.GetDownloadRecordsFiltered(dbFilter)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"records": records,
		"total":   total,
	}, nil
}

// DeleteHistoryRecord deletes a single download history record
func (a *App) DeleteHistoryRecord(id int64) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return a.db.DeleteDownloadRecord(id)
}

// ClearDownloadHistory removes all download history
func (a *App) ClearDownloadHistory() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	err := a.db.ClearAllHistory()
	if err == nil && a.logBuffer != nil {
		a.logBuffer.Info("Download history cleared")
	}
	return err
}

// RefetchFromHistory re-downloads content from history
func (a *App) RefetchFromHistory(tidalContentID string) (map[string]interface{}, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Get the record to find the content type
	record, err := a.db.GetDownloadRecord(tidalContentID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, fmt.Errorf("history record not found")
	}

	// Reconstruct the Tidal URL
	var url string
	switch record.ContentType {
	case "playlist":
		url = fmt.Sprintf("https://tidal.com/browse/playlist/%s", tidalContentID)
	case "album":
		url = fmt.Sprintf("https://tidal.com/browse/album/%s", tidalContentID)
	case "track":
		url = fmt.Sprintf("https://tidal.com/browse/track/%s", tidalContentID)
	default:
		return nil, fmt.Errorf("unknown content type: %s", record.ContentType)
	}

	// Fetch the content
	return a.FetchTidalContent(url)
}

// GetMatchFailures returns all match failures
func (a *App) GetMatchFailures() ([]core.MatchFailure, error) {
	if a.db == nil {
		return nil, nil
	}
	return a.db.GetMatchFailures()
}
