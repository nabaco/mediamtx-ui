package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const schema = `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS users (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    username           TEXT    UNIQUE NOT NULL,
    password_hash      TEXT    NOT NULL,
    stream_token_hash  TEXT,
    role               TEXT    NOT NULL DEFAULT 'user',
    enabled            INTEGER NOT NULL DEFAULT 1,
    created_at         DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at         DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS groups (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_groups (
    user_id  INTEGER NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    group_id INTEGER NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, group_id)
);

CREATE TABLE IF NOT EXISTS acls (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    subject_type   TEXT    NOT NULL CHECK (subject_type IN ('user','group')),
    subject_id     INTEGER NOT NULL,
    stream_pattern TEXT    NOT NULL,
    action         TEXT    NOT NULL CHECK (action IN ('read','publish','both')),
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS stream_metadata (
    path_name   TEXT PRIMARY KEY,
    description TEXT NOT NULL DEFAULT '',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS audit_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    username    TEXT    NOT NULL,
    stream_path TEXT    NOT NULL,
    action      TEXT    NOT NULL,
    protocol    TEXT    NOT NULL DEFAULT '',
    remote_addr TEXT    NOT NULL DEFAULT '',
    allowed     INTEGER NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON audit_log (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_username   ON audit_log (username);
CREATE INDEX IF NOT EXISTS idx_acls_subject         ON acls (subject_type, subject_id);
`

// Open opens (or creates) the SQLite database at path and runs schema migrations.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}

	// Single writer to avoid SQLITE_BUSY under WAL
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("run schema: %w", err)
	}

	return db, nil
}
