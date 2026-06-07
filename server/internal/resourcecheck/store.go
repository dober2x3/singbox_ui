package resourcecheck

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// Store persists check results to SQLite.
type Store struct {
	db *sqlx.DB
}

// NewStore opens (or creates) the SQLite database and runs migrations.
func NewStore(dbPath string) (*Store, error) {
	db, err := sqlx.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{db: db}, nil
}

func migrate(db *sqlx.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS tags (
		tag TEXT PRIMARY KEY
	);
	CREATE TABLE IF NOT EXISTS check_results (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		resource    TEXT NOT NULL,
		tag         TEXT NOT NULL,
		status      TEXT NOT NULL,
		latency_ms  INTEGER,
		http_code   INTEGER,
		error       TEXT,
		checked_at  TEXT NOT NULL,
		FOREIGN KEY (tag) REFERENCES tags(tag)
	);
	CREATE INDEX IF NOT EXISTS idx_results_tag_resource ON check_results(tag, resource);
	CREATE INDEX IF NOT EXISTS idx_results_resource ON check_results(resource);
	CREATE INDEX IF NOT EXISTS idx_results_checked_at ON check_results(checked_at);
	CREATE VIEW IF NOT EXISTS latest_results AS
	SELECT cr.* FROM check_results cr
	INNER JOIN (
		SELECT resource, tag, MAX(id) as max_id
		FROM check_results
		GROUP BY resource, tag
	) latest ON cr.id = latest.max_id;
	`
	_, err := db.Exec(schema)
	return err
}

// SaveResult inserts a check result and ensures the tag exists.
func (s *Store) SaveResult(r CheckResult) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Upsert tag
	_, err = tx.Exec("INSERT OR IGNORE INTO tags (tag) VALUES (?)", r.Tag)
	if err != nil {
		return fmt.Errorf("insert tag: %w", err)
	}

	// Insert result
	_, err = tx.Exec(
		`INSERT INTO check_results (resource, tag, status, latency_ms, http_code, error, checked_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.Resource, r.Tag, r.Status, r.LatencyMs, r.HTTPCode, r.Error, r.CheckedAt,
	)
	if err != nil {
		return fmt.Errorf("insert result: %w", err)
	}

	return tx.Commit()
}

// GetLatestResults returns the most recent result for each (resource, tag) pair.
func (s *Store) GetLatestResults() ([]CheckResult, error) {
	var results []CheckResult
	err := s.db.Select(&results, `SELECT * FROM latest_results ORDER BY tag, resource`)
	return results, err
}

// GetResultsForTag returns the latest results for a specific node tag.
func (s *Store) GetResultsForTag(tag string) ([]CheckResult, error) {
	var results []CheckResult
	err := s.db.Select(&results,
		`SELECT * FROM latest_results WHERE tag = ? ORDER BY resource`, tag)
	return results, err
}

// GetHistory returns check history for a (resource, tag) pair, newest first.
func (s *Store) GetHistory(resource, tag string, limit int) ([]CheckResult, error) {
	if limit <= 0 {
		limit = 50
	}
	var results []CheckResult
	err := s.db.Select(&results,
		`SELECT * FROM check_results WHERE resource = ? AND tag = ?
		 ORDER BY id DESC LIMIT ?`, resource, tag, limit)
	return results, err
}

// GetTags returns all known proxy node tags.
func (s *Store) GetTags() ([]string, error) {
	var tags []string
	err := s.db.Select(&tags, "SELECT tag FROM tags ORDER BY tag")
	return tags, err
}

// Close shuts down the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
