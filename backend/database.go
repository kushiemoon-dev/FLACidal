package backend

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Database wraps the SQLite connection
type Database struct {
	db *sql.DB
}

// NewDatabase creates and initializes the database
func NewDatabase() (*Database, error) {
	if err := EnsureDataDir(); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", GetDatabasePath())
	if err != nil {
		return nil, err
	}

	database := &Database{db: db}
	if err := database.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return database, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// migrate creates the database schema
func (d *Database) migrate() error {
	schema := `
	-- Download history: tracks downloaded playlists/albums
	CREATE TABLE IF NOT EXISTS download_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tidal_content_id TEXT NOT NULL,
		tidal_content_name TEXT,
		content_type TEXT,  -- 'playlist', 'album', 'track'
		last_download_at DATETIME,
		tracks_total INTEGER DEFAULT 0,
		tracks_downloaded INTEGER DEFAULT 0,
		tracks_failed INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(tidal_content_id)
	);

	-- Track cache: maps ISRC to track info for matching
	CREATE TABLE IF NOT EXISTS track_cache (
		isrc TEXT PRIMARY KEY,
		tidal_track_id TEXT,
		spotify_track_id TEXT,
		spotify_uri TEXT,
		title TEXT,
		artist TEXT,
		match_method TEXT,  -- 'isrc', 'fuzzy'
		confidence REAL DEFAULT 1.0,
		matched_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Match failures: tracks that couldn't be matched
	CREATE TABLE IF NOT EXISTS match_failures (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tidal_track_id TEXT NOT NULL,
		isrc TEXT,
		title TEXT,
		artist TEXT,
		album TEXT,
		reason TEXT,
		attempts INTEGER DEFAULT 1,
		last_attempt_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(tidal_track_id)
	);

	-- Create indexes for faster lookups
	CREATE INDEX IF NOT EXISTS idx_track_cache_spotify ON track_cache(spotify_track_id);
	CREATE INDEX IF NOT EXISTS idx_download_history_tidal ON download_history(tidal_content_id);
	`

	_, err := d.db.Exec(schema)
	return err
}

// =============================================================================
// Track Cache Operations
// =============================================================================

// CachedTrack represents a cached track mapping
type CachedTrack struct {
	ISRC           string    `json:"isrc"`
	TidalTrackID   string    `json:"tidalTrackId"`
	SpotifyTrackID string    `json:"spotifyTrackId"`
	SpotifyURI     string    `json:"spotifyUri"`
	Title          string    `json:"title"`
	Artist         string    `json:"artist"`
	MatchMethod    string    `json:"matchMethod"`
	Confidence     float64   `json:"confidence"`
	MatchedAt      time.Time `json:"matchedAt"`
}

// GetCachedTrack retrieves a track from cache by ISRC
func (d *Database) GetCachedTrack(isrc string) (*CachedTrack, error) {
	row := d.db.QueryRow(`
		SELECT isrc, tidal_track_id, spotify_track_id, spotify_uri,
		       title, artist, match_method, confidence, matched_at
		FROM track_cache WHERE isrc = ?
	`, isrc)

	var track CachedTrack
	err := row.Scan(
		&track.ISRC, &track.TidalTrackID, &track.SpotifyTrackID, &track.SpotifyURI,
		&track.Title, &track.Artist, &track.MatchMethod, &track.Confidence, &track.MatchedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &track, nil
}

// CacheTrack saves a track mapping to cache
func (d *Database) CacheTrack(track *CachedTrack) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO track_cache
		(isrc, tidal_track_id, spotify_track_id, spotify_uri, title, artist, match_method, confidence, matched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		track.ISRC, track.TidalTrackID, track.SpotifyTrackID, track.SpotifyURI,
		track.Title, track.Artist, track.MatchMethod, track.Confidence, time.Now(),
	)
	return err
}

// GetCacheStats returns cache statistics
func (d *Database) GetCacheStats() (total int, byMethod map[string]int, err error) {
	byMethod = make(map[string]int)

	// Total count
	row := d.db.QueryRow("SELECT COUNT(*) FROM track_cache")
	if err = row.Scan(&total); err != nil {
		return
	}

	// Count by method
	rows, err := d.db.Query("SELECT match_method, COUNT(*) FROM track_cache GROUP BY match_method")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var method string
		var count int
		if err = rows.Scan(&method, &count); err != nil {
			return
		}
		byMethod[method] = count
	}

	return
}

// =============================================================================
// Download History Operations
// =============================================================================

// DownloadRecord represents a download history entry
type DownloadRecord struct {
	ID               int64     `json:"id"`
	TidalContentID   string    `json:"tidalContentId"`
	TidalContentName string    `json:"tidalContentName"`
	ContentType      string    `json:"contentType"`
	LastDownloadAt   time.Time `json:"lastDownloadAt"`
	TracksTotal      int       `json:"tracksTotal"`
	TracksDownloaded int       `json:"tracksDownloaded"`
	TracksFailed     int       `json:"tracksFailed"`
	CreatedAt        time.Time `json:"createdAt"`
}

// GetDownloadRecord retrieves download history for a Tidal content
func (d *Database) GetDownloadRecord(tidalContentID string) (*DownloadRecord, error) {
	row := d.db.QueryRow(`
		SELECT id, tidal_content_id, tidal_content_name, content_type,
		       last_download_at, tracks_total, tracks_downloaded,
		       tracks_failed, created_at
		FROM download_history WHERE tidal_content_id = ?
	`, tidalContentID)

	var record DownloadRecord
	var lastDownloadAt, createdAt sql.NullTime
	err := row.Scan(
		&record.ID, &record.TidalContentID, &record.TidalContentName,
		&record.ContentType, &lastDownloadAt, &record.TracksTotal,
		&record.TracksDownloaded, &record.TracksFailed, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastDownloadAt.Valid {
		record.LastDownloadAt = lastDownloadAt.Time
	}
	if createdAt.Valid {
		record.CreatedAt = createdAt.Time
	}
	return &record, nil
}

// SaveDownloadRecord creates or updates a download record
func (d *Database) SaveDownloadRecord(record *DownloadRecord) error {
	_, err := d.db.Exec(`
		INSERT INTO download_history
		(tidal_content_id, tidal_content_name, content_type,
		 last_download_at, tracks_total, tracks_downloaded, tracks_failed)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tidal_content_id) DO UPDATE SET
			tidal_content_name = excluded.tidal_content_name,
			content_type = excluded.content_type,
			last_download_at = excluded.last_download_at,
			tracks_total = excluded.tracks_total,
			tracks_downloaded = excluded.tracks_downloaded,
			tracks_failed = excluded.tracks_failed
	`,
		record.TidalContentID, record.TidalContentName, record.ContentType,
		time.Now(), record.TracksTotal, record.TracksDownloaded, record.TracksFailed,
	)
	return err
}

// HistoryFilter contains filtering options for download history
type HistoryFilter struct {
	ContentType string    `json:"contentType,omitempty"` // "playlist", "album", "track" or empty for all
	DateFrom    time.Time `json:"dateFrom,omitempty"`
	DateTo      time.Time `json:"dateTo,omitempty"`
	Search      string    `json:"search,omitempty"`
	Limit       int       `json:"limit,omitempty"`
	Offset      int       `json:"offset,omitempty"`
}

// GetDownloadRecordsFiltered returns filtered download history with pagination
func (d *Database) GetDownloadRecordsFiltered(filter HistoryFilter) ([]DownloadRecord, int, error) {
	// Build WHERE clause
	where := "1=1"
	args := []interface{}{}

	if filter.ContentType != "" {
		where += " AND content_type = ?"
		args = append(args, filter.ContentType)
	}

	if !filter.DateFrom.IsZero() {
		where += " AND last_download_at >= ?"
		args = append(args, filter.DateFrom)
	}

	if !filter.DateTo.IsZero() {
		where += " AND last_download_at <= ?"
		args = append(args, filter.DateTo)
	}

	if filter.Search != "" {
		where += " AND (tidal_content_name LIKE ? OR tidal_content_id LIKE ?)"
		searchTerm := "%" + filter.Search + "%"
		args = append(args, searchTerm, searchTerm)
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM download_history WHERE " + where
	if err := d.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build main query with pagination
	query := `
		SELECT id, tidal_content_id, tidal_content_name, content_type,
		       last_download_at, tracks_total, tracks_downloaded,
		       tracks_failed, created_at
		FROM download_history WHERE ` + where + `
		ORDER BY last_download_at DESC`

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var records []DownloadRecord
	for rows.Next() {
		var record DownloadRecord
		var lastDownloadAt, createdAt sql.NullTime
		if err := rows.Scan(
			&record.ID, &record.TidalContentID, &record.TidalContentName,
			&record.ContentType, &lastDownloadAt, &record.TracksTotal,
			&record.TracksDownloaded, &record.TracksFailed, &createdAt,
		); err != nil {
			return nil, 0, err
		}
		if lastDownloadAt.Valid {
			record.LastDownloadAt = lastDownloadAt.Time
		}
		if createdAt.Valid {
			record.CreatedAt = createdAt.Time
		}
		records = append(records, record)
	}
	return records, total, nil
}

// DeleteDownloadRecord removes a single download record by ID
func (d *Database) DeleteDownloadRecord(id int64) error {
	result, err := d.db.Exec("DELETE FROM download_history WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ClearAllHistory removes all download history records
func (d *Database) ClearAllHistory() error {
	_, err := d.db.Exec("DELETE FROM download_history")
	return err
}

// GetAllDownloadRecords returns all download history
func (d *Database) GetAllDownloadRecords() ([]DownloadRecord, error) {
	rows, err := d.db.Query(`
		SELECT id, tidal_content_id, tidal_content_name, content_type,
		       last_download_at, tracks_total, tracks_downloaded,
		       tracks_failed, created_at
		FROM download_history ORDER BY last_download_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []DownloadRecord
	for rows.Next() {
		var record DownloadRecord
		var lastDownloadAt, createdAt sql.NullTime
		if err := rows.Scan(
			&record.ID, &record.TidalContentID, &record.TidalContentName,
			&record.ContentType, &lastDownloadAt, &record.TracksTotal,
			&record.TracksDownloaded, &record.TracksFailed, &createdAt,
		); err != nil {
			return nil, err
		}
		if lastDownloadAt.Valid {
			record.LastDownloadAt = lastDownloadAt.Time
		}
		if createdAt.Valid {
			record.CreatedAt = createdAt.Time
		}
		records = append(records, record)
	}
	return records, nil
}

// =============================================================================
// Match Failures Operations
// =============================================================================

// MatchFailure represents a track that couldn't be matched
type MatchFailure struct {
	ID            int64     `json:"id"`
	TidalTrackID  string    `json:"tidalTrackId"`
	ISRC          string    `json:"isrc"`
	Title         string    `json:"title"`
	Artist        string    `json:"artist"`
	Album         string    `json:"album"`
	Reason        string    `json:"reason"`
	Attempts      int       `json:"attempts"`
	LastAttemptAt time.Time `json:"lastAttemptAt"`
}

// RecordMatchFailure saves or updates a match failure
func (d *Database) RecordMatchFailure(failure *MatchFailure) error {
	_, err := d.db.Exec(`
		INSERT INTO match_failures
		(tidal_track_id, isrc, title, artist, album, reason, attempts, last_attempt_at)
		VALUES (?, ?, ?, ?, ?, ?, 1, ?)
		ON CONFLICT(tidal_track_id) DO UPDATE SET
			reason = excluded.reason,
			attempts = attempts + 1,
			last_attempt_at = excluded.last_attempt_at
	`,
		failure.TidalTrackID, failure.ISRC, failure.Title,
		failure.Artist, failure.Album, failure.Reason, time.Now(),
	)
	return err
}

// GetMatchFailures returns all match failures
func (d *Database) GetMatchFailures() ([]MatchFailure, error) {
	rows, err := d.db.Query(`
		SELECT id, tidal_track_id, isrc, title, artist, album, reason, attempts, last_attempt_at
		FROM match_failures ORDER BY last_attempt_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var failures []MatchFailure
	for rows.Next() {
		var f MatchFailure
		if err := rows.Scan(
			&f.ID, &f.TidalTrackID, &f.ISRC, &f.Title, &f.Artist,
			&f.Album, &f.Reason, &f.Attempts, &f.LastAttemptAt,
		); err != nil {
			return nil, err
		}
		failures = append(failures, f)
	}
	return failures, nil
}

// ClearMatchFailure removes a failure (when retry succeeds)
func (d *Database) ClearMatchFailure(tidalTrackID string) error {
	_, err := d.db.Exec("DELETE FROM match_failures WHERE tidal_track_id = ?", tidalTrackID)
	return err
}

// GetFailureCount returns the number of failed matches
func (d *Database) GetFailureCount() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM match_failures").Scan(&count)
	return count, err
}
