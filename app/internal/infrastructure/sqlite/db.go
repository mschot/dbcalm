package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS user (
	username TEXT PRIMARY KEY,
	password TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS client (
	id TEXT PRIMARY KEY,
	secret TEXT NOT NULL,
	label TEXT NOT NULL,
	scopes TEXT NOT NULL, -- JSON array
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS auth_code (
	code TEXT PRIMARY KEY,
	username TEXT NOT NULL,
	scopes TEXT NOT NULL, -- JSON array
	expires_at DATETIME NOT NULL,
	created_at DATETIME NOT NULL,
	FOREIGN KEY (username) REFERENCES user(username) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS schedule (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	backup_type TEXT NOT NULL,
	frequency TEXT NOT NULL,
	day_of_week INTEGER,
	day_of_month INTEGER,
	hour INTEGER,
	minute INTEGER,
	interval_value INTEGER,
	interval_unit TEXT,
	retention_value INTEGER,
	retention_unit TEXT,
	enabled INTEGER NOT NULL DEFAULT 1,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS process (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	command_id TEXT NOT NULL,
	command TEXT NOT NULL,
	pid INTEGER,
	status TEXT NOT NULL,
	output TEXT,
	error TEXT,
	return_code INTEGER,
	start_time DATETIME NOT NULL,
	end_time DATETIME,
	type TEXT NOT NULL,
	args TEXT NOT NULL -- JSON object
);

CREATE TABLE IF NOT EXISTS backup (
	id TEXT PRIMARY KEY,
	from_backup_id TEXT,
	schedule_id INTEGER,
	start_time DATETIME NOT NULL,
	end_time DATETIME,
	process_id INTEGER NOT NULL,
	size INTEGER,
	FOREIGN KEY (from_backup_id) REFERENCES backup(id) ON DELETE CASCADE,
	FOREIGN KEY (schedule_id) REFERENCES schedule(id) ON DELETE SET NULL,
	FOREIGN KEY (process_id) REFERENCES process(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS restore (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	backup_id TEXT NOT NULL,
	backup_timestamp DATETIME NOT NULL,
	target TEXT NOT NULL,
	target_path TEXT NOT NULL,
	start_time DATETIME NOT NULL,
	end_time DATETIME,
	process_id INTEGER NOT NULL,
	FOREIGN KEY (backup_id) REFERENCES backup(id) ON DELETE CASCADE,
	FOREIGN KEY (process_id) REFERENCES process(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_backups_schedule_id ON backup(schedule_id);
CREATE INDEX IF NOT EXISTS idx_backups_start_time ON backup(start_time);
CREATE INDEX IF NOT EXISTS idx_processes_status ON process(status);
CREATE INDEX IF NOT EXISTS idx_processes_type ON process(type);
CREATE INDEX IF NOT EXISTS idx_processes_command_id ON process(command_id);
CREATE INDEX IF NOT EXISTS idx_auth_codes_expires_at ON auth_code(expires_at);
`

type DB struct {
	*sqlx.DB
}

func New(dbPath string) (*DB, error) {
	db, err := sqlx.Connect("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable WAL mode for better concurrency (allows concurrent reads/writes)
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Set busy timeout to handle concurrent access from multiple services
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create tables
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

// NullString helper for optional string fields
func NullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

// NullInt64 helper for optional int64 fields
func NullInt64(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}

// NullInt helper for optional int fields
func NullInt(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
}

// NullTime helper for optional time fields
func NullTime(t *interface{}) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	// This will be properly handled by the specific repository
	return sql.NullTime{Valid: false}
}
