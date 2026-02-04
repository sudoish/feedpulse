package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// FeedItem represents a normalized feed item
type FeedItem struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Source    string    `json:"source"`
	Timestamp *string   `json:"timestamp,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	RawData   *string   `json:"raw_data,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// FetchLog represents a fetch operation log entry
type FetchLog struct {
	ID           int
	Source       string
	FetchedAt    time.Time
	Status       string
	ItemsCount   int
	ErrorMessage *string
	DurationMs   int64
}

// FetchStats represents statistics for a feed source
type FetchStats struct {
	Source       string
	ItemsCount   int
	ErrorCount   int
	TotalFetches int
	LastSuccess  *string
}

// Storage handles database operations
type Storage struct {
	db *sql.DB
}

// NewStorage creates a new storage instance
func NewStorage(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Set busy timeout
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	s := &Storage{db: db}

	// Initialize schema
	if err := s.initSchema(); err != nil {
		return nil, err
	}

	return s, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// initSchema creates tables and indexes if they don't exist
func (s *Storage) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS feed_items (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    url TEXT NOT NULL,
    source TEXT NOT NULL,
    timestamp TEXT,
    tags TEXT,
    raw_data TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS fetch_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source TEXT NOT NULL,
    fetched_at TEXT NOT NULL,
    status TEXT NOT NULL,
    items_count INTEGER DEFAULT 0,
    error_message TEXT,
    duration_ms INTEGER
);

CREATE INDEX IF NOT EXISTS idx_feed_items_source ON feed_items(source);
CREATE INDEX IF NOT EXISTS idx_feed_items_timestamp ON feed_items(timestamp);
CREATE INDEX IF NOT EXISTS idx_fetch_log_source ON fetch_log(source);
`

	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}

// SaveItems saves feed items in a transaction
func (s *Storage) SaveItems(items []FeedItem) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO feed_items (id, title, url, source, timestamp, tags, raw_data, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			url = excluded.url,
			timestamp = excluded.timestamp,
			tags = excluded.tags,
			raw_data = excluded.raw_data
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, item := range items {
		// Serialize tags as JSON
		var tagsJSON *string
		if len(item.Tags) > 0 {
			tagsBytes, err := json.Marshal(item.Tags)
			if err != nil {
				return fmt.Errorf("failed to marshal tags: %w", err)
			}
			tagsStr := string(tagsBytes)
			tagsJSON = &tagsStr
		}

		_, err := stmt.Exec(
			item.ID,
			item.Title,
			item.URL,
			item.Source,
			item.Timestamp,
			tagsJSON,
			item.RawData,
			item.CreatedAt.Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("failed to insert item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// LogFetch logs a fetch operation
func (s *Storage) LogFetch(log FetchLog) error {
	_, err := s.db.Exec(`
		INSERT INTO fetch_log (source, fetched_at, status, items_count, error_message, duration_ms)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		log.Source,
		log.FetchedAt.Format(time.RFC3339),
		log.Status,
		log.ItemsCount,
		log.ErrorMessage,
		log.DurationMs,
	)
	if err != nil {
		return fmt.Errorf("failed to log fetch: %w", err)
	}

	return nil
}

// GetItemCount returns the total number of items for a source
func (s *Storage) GetItemCount(source string) (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM feed_items WHERE source = ?", source).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get item count: %w", err)
	}
	return count, nil
}

// GetAllItemsCount returns the total number of items across all sources
func (s *Storage) GetAllItemsCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM feed_items").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get total item count: %w", err)
	}
	return count, nil
}

// GetFetchStats returns fetch statistics for all sources
func (s *Storage) GetFetchStats() ([]FetchStats, error) {
	query := `
		WITH source_stats AS (
			SELECT 
				source,
				COUNT(*) as total_fetches,
				SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as error_count,
				MAX(CASE WHEN status = 'success' THEN fetched_at ELSE NULL END) as last_success
			FROM fetch_log
			GROUP BY source
		)
		SELECT 
			COALESCE(ss.source, fi.source) as source,
			COUNT(DISTINCT fi.id) as items_count,
			COALESCE(ss.error_count, 0) as error_count,
			COALESCE(ss.total_fetches, 0) as total_fetches,
			ss.last_success
		FROM feed_items fi
		LEFT JOIN source_stats ss ON fi.source = ss.source
		GROUP BY fi.source
		UNION
		SELECT 
			source,
			0 as items_count,
			error_count,
			total_fetches,
			last_success
		FROM source_stats
		WHERE source NOT IN (SELECT DISTINCT source FROM feed_items)
		ORDER BY source
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query fetch stats: %w", err)
	}
	defer rows.Close()

	var stats []FetchStats
	for rows.Next() {
		var stat FetchStats
		var lastSuccess *string

		err := rows.Scan(
			&stat.Source,
			&stat.ItemsCount,
			&stat.ErrorCount,
			&stat.TotalFetches,
			&lastSuccess,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if lastSuccess != nil {
			stat.LastSuccess = lastSuccess
		}

		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return stats, nil
}
