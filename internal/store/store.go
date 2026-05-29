// Package store persists monitoring events to a local SQLite database and
// answers the aggregation queries the reports and status bar need.
//
// It uses the pure-Go modernc.org/sqlite driver so worklog builds without cgo
// and ships as a single binary for both Apple Silicon and Intel.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	// Register the pure-Go "sqlite" driver for database/sql (no cgo).
	_ "modernc.org/sqlite"
)

// Store wraps the SQLite database connection.
type Store struct {
	db *sql.DB
}

// Event is one monitoring sample, written roughly every poll interval.
type Event struct {
	ID       int64
	TS       int64 // Unix seconds
	App      string
	Title    string
	IsIdle   bool
	Project  string
	Category string
}

const schema = `
CREATE TABLE IF NOT EXISTS events (
  id       INTEGER PRIMARY KEY,
  ts       INTEGER NOT NULL,
  app      TEXT NOT NULL,
  title    TEXT NOT NULL,
  is_idle  INTEGER NOT NULL DEFAULT 0,
  project  TEXT,
  category TEXT
);
CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts);
CREATE INDEX IF NOT EXISTS idx_events_project ON events(project);
`

// Open opens (creating if needed) the database at path and applies the schema.
// Parent directories are created automatically.
func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create data dir: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	// A single writer (the daemon) plus occasional readers; WAL keeps the
	// status bar from blocking the daemon.
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("set pragmas: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

// Insert writes a single event.
func (s *Store) Insert(e Event) error {
	idle := 0
	if e.IsIdle {
		idle = 1
	}
	_, err := s.db.Exec(
		`INSERT INTO events (ts, app, title, is_idle, project, category) VALUES (?, ?, ?, ?, ?, ?)`,
		e.TS, e.App, e.Title, idle, e.Project, e.Category,
	)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	return nil
}

// Events returns all events with ts in [from, to), ordered by ts ascending.
func (s *Store) Events(from, to int64) ([]Event, error) {
	rows, err := s.db.Query(
		`SELECT id, ts, app, title, is_idle, COALESCE(project,''), COALESCE(category,'')
		 FROM events WHERE ts >= ? AND ts < ? ORDER BY ts ASC`,
		from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var e Event
		var idle int
		if err := rows.Scan(&e.ID, &e.TS, &e.App, &e.Title, &idle, &e.Project, &e.Category); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		e.IsIdle = idle != 0
		out = append(out, e)
	}
	return out, rows.Err()
}

// Wipe deletes events with ts < before. Use before<=0 with All to clear all.
func (s *Store) Wipe(before int64) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM events WHERE ts < ?`, before)
	if err != nil {
		return 0, fmt.Errorf("wipe events: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// WipeAll deletes every event.
func (s *Store) WipeAll() (int64, error) {
	res, err := s.db.Exec(`DELETE FROM events`)
	if err != nil {
		return 0, fmt.Errorf("wipe all: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}
