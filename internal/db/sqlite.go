package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS connections (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  tracker_kind TEXT NOT NULL,
  base_url TEXT NOT NULL,
  keyring_key TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS templates (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  tracker_type_name TEXT NOT NULL,
  fields_schema TEXT NOT NULL,
  ai_instructions TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS logs (
  id TEXT PRIMARY KEY,
  timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
  action TEXT NOT NULL,
  connection_id TEXT,
  template_id TEXT,
  raw_input TEXT,
  generated_content TEXT,
  final_content TEXT,
  result_ticket_id TEXT,
  result_ticket_url TEXT,
  status TEXT NOT NULL,
  error_message TEXT
);

CREATE TABLE IF NOT EXISTS ai_settings (
  id TEXT PRIMARY KEY DEFAULT 'default',
  base_url TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  keyring_key TEXT NOT NULL DEFAULT '',
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS generate_prefs (
  id TEXT PRIMARY KEY DEFAULT 'default',
  connection_id TEXT NOT NULL DEFAULT '',
  project_id TEXT NOT NULL DEFAULT '',
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS notes (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  content TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  merged_from_ids TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

// DefaultPath returns the OS-appropriate path to the Ticketsmith SQLite
// database file, creating its parent directory if necessary.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	appDir := filepath.Join(dir, "ticketsmith")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return "", fmt.Errorf("create app data dir: %w", err)
	}
	return filepath.Join(appDir, "ticketsmith.db"), nil
}

// Open opens (creating if necessary) the SQLite database at path and ensures
// the schema exists. Safe to call repeatedly.
func Open(path string) (*sql.DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	// Single-user desktop app: serialize at the pool level rather than
	// tuning SQLite WAL/busy-timeout behavior.
	sqlDB.SetMaxOpenConns(1)

	if _, err := sqlDB.Exec(schema); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return sqlDB, nil
}
