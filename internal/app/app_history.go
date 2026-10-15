package app

import (
	"fmt"
	"sort"
	"strings"
	"time"

	core "github.com/kushiemoon-dev/flacidal-core"
)

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

func (a *App) GetDownloadHistory() ([]core.DownloadRecord, error) {
	if a.db == nil {
		return nil, nil
	}
	return a.db.GetAllDownloadRecords()
}

// TrackHistoryPage backs ListTrackHistory and GET /api/track-history. status
// is "", "failed" or "completed"; limit is capped to 1..200 (default 50).
func TrackHistoryPage(db *core.Database, status string, limit, offset int) (map[string]interface{}, error) {
	if status != "" && status != "failed" && status != "completed" {
		return nil, fmt.Errorf("invalid status %q", status)
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	entries, err := db.ListHistoryByStatus(status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("ListTrackHistory: %w", err)
	}
	total, err := db.GetHistoryCountByStatus(status)
	if err != nil {
		return nil, fmt.Errorf("ListTrackHistory: %w", err)
	}
	return map[string]interface{}{"entries": entries, "total": total}, nil
}

// ListTrackHistory lists per-track download history, newest first. status is
// "", "failed" or "completed".
func (a *App) ListTrackHistory(status string, limit, offset int) (map[string]interface{}, error) {
	if a.db == nil {
		return map[string]interface{}{"entries": []core.HistoryEntry{}, "total": 0}, nil
	}
	return TrackHistoryPage(a.db, status, limit, offset)
}

// Deduplicated, for the home page grid.
func (a *App) GetRecentAlbums(limit int) ([]map[string]interface{}, error) {
	if a.db == nil {
		return []map[string]interface{}{}, nil
	}
	return RecentAlbums(a.db, limit)
}

// RecentAlbums backs GetRecentAlbums and is shared by both the desktop
// (Wails) and HTTP server APIs, the same sharing pattern used by
// ConvertTidalSearchResults / SearchDeezerTracks in app_search.go.
func RecentAlbums(db *core.Database, limit int) ([]map[string]interface{}, error) {
	records, err := db.GetAllDownloadRecords()
	if err != nil {
		return nil, fmt.Errorf("GetRecentAlbums: %w", err)
	}

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

func (a *App) GetDownloadHistoryFiltered(filter map[string]interface{}) (map[string]interface{}, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

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

func (a *App) DeleteHistoryRecord(id int64) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return a.db.DeleteDownloadRecord(id)
}

func (a *App) ClearDownloadHistory() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	err := a.db.ClearAllHistory()
	if err == nil && a.logBuffer != nil {
		a.logBuffer.Info("Cleared the download history")
	}
	return err
}

func (a *App) RefetchFromHistory(tidalContentID string) (map[string]interface{}, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	record, err := a.db.GetDownloadRecord(tidalContentID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, fmt.Errorf("history record not found")
	}

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

	return a.FetchTidalContent(url)
}

func (a *App) GetMatchFailures() ([]core.MatchFailure, error) {
	if a.db == nil {
		return nil, nil
	}
	return a.db.GetMatchFailures()
}
