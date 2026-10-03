package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func Open(dbPath string) (*DB, error) {
	// Enable WAL mode, foreign keys, and busy timeout for concurrent safety
	dsn := dbPath
	if !strings.Contains(dsn, "?") {
		dsn += "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)"
	}

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	database := &DB{sqlDB}
	if err := database.Migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return database, nil
}

func (d *DB) Migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS boards (
		id TEXT PRIMARY KEY,
		tv_secret_hash TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS join_tokens (
		token_hash TEXT PRIMARY KEY,
		board_id TEXT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
		expires_at DATETIME NOT NULL,
		used_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS devices (
		id TEXT PRIMARY KEY,
		board_id TEXT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
		session_hash TEXT NOT NULL UNIQUE,
		label TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		last_seen DATETIME NOT NULL,
		revoked_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS items (
		id TEXT PRIMARY KEY,
		board_id TEXT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
		type TEXT NOT NULL,
		text TEXT NOT NULL,
		when_ts DATETIME,
		done INTEGER NOT NULL DEFAULT 0,
		archived INTEGER NOT NULL DEFAULT 0,
		created_by_device TEXT REFERENCES devices(id),
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_items_board_archived ON items(board_id, archived);
	CREATE INDEX IF NOT EXISTS idx_devices_session ON devices(session_hash);
	CREATE INDEX IF NOT EXISTS idx_join_tokens_hash ON join_tokens(token_hash);
	CREATE INDEX IF NOT EXISTS idx_join_tokens_board ON join_tokens(board_id);
	`
	_, err := d.Exec(schema)
	return err
}
