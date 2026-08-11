package db

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type AuditEntry struct {
	ID         int64
	Username   string
	StreamPath string
	Action     string
	Protocol   string
	RemoteAddr string
	Allowed    bool
	CreatedAt  time.Time
}

func WriteAuditLog(db *sql.DB, e AuditEntry) error {
	allowed := 0
	if e.Allowed {
		allowed = 1
	}
	_, err := db.Exec(
		`INSERT INTO audit_log (username, stream_path, action, protocol, remote_addr, allowed)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		e.Username, e.StreamPath, e.Action, e.Protocol, e.RemoteAddr, allowed,
	)
	return err
}

// PruneAuditLog deletes audit rows older than retentionDays and returns how
// many went. Every mediamtx auth callback writes a row here — a dead camera
// source retrying, or any reconnect loop, produces tens of thousands of rows a
// day, so the table has to be capped or it eventually dwarfs the real data
// (530 MB / 3.7M rows by 2026-08-11 on tec14).
//
// Deletion is batched: the pool is capped at one connection, so a single
// multi-million-row DELETE would hold the only writer for the whole purge and
// block every auth callback behind it.
func PruneAuditLog(db *sql.DB, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}
	cutoff := fmt.Sprintf("-%d days", retentionDays)

	var total int64
	for {
		res, err := db.Exec(
			`DELETE FROM audit_log WHERE id IN (
			   SELECT id FROM audit_log WHERE created_at < datetime('now', ?) LIMIT 5000
			 )`, cutoff)
		if err != nil {
			return total, fmt.Errorf("prune audit_log: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return total, err
		}
		total += n
		if n == 0 {
			return total, nil
		}
	}
}

// VacuumDB rewrites the database file to give freed pages back to the
// filesystem. Only worth calling after a large prune — SQLite reuses free pages
// on its own, so routine daily deletes need no vacuum.
//
// VACUUM builds the replacement in a temporary file first, and the container
// image has no writable /tmp: the default search (SQLITE_TMPDIR, TMPDIR,
// /var/tmp, /usr/tmp, /tmp) comes up empty and VACUUM fails with
// SQLITE_IOERR_GETTEMPPATH (6410). Setting the env var from Go does not help —
// modernc.org/sqlite is a translation of the C library and its getenv reads an
// environ snapshot taken at process start. temp_store_directory sets the same
// value through the API, so it works in-process, and it is checked before the
// env vars. It is marked deprecated upstream but still honoured.
func VacuumDB(db *sql.DB) error {
	var seq int
	var name, file string
	if err := db.QueryRow(`SELECT seq, name, file FROM pragma_database_list WHERE name = 'main'`).
		Scan(&seq, &name, &file); err != nil {
		return fmt.Errorf("vacuum: locate database file: %w", err)
	}
	if dir := filepath.Dir(file); file != "" && dir != "" {
		// No placeholders in PRAGMA statements; the value is the path SQLite
		// itself just reported, not user input.
		if _, err := db.Exec(fmt.Sprintf("PRAGMA temp_store_directory = '%s'", strings.ReplaceAll(dir, "'", "''"))); err != nil {
			return fmt.Errorf("vacuum: set temp dir %s: %w", dir, err)
		}
	}
	if _, err := db.Exec("VACUUM"); err != nil {
		return fmt.Errorf("vacuum: %w", err)
	}
	// Under WAL the rebuilt database lands in the -wal file; the main file only
	// shrinks once those frames are checkpointed back and the log is truncated.
	// Without this the vacuum "succeeds" and the file on disk never changes.
	if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return fmt.Errorf("vacuum: checkpoint: %w", err)
	}
	return nil
}

type AuditFilter struct {
	Username   string
	StreamPath string
	Limit      int
	Offset     int
}

func ListAuditLog(db *sql.DB, f AuditFilter) ([]*AuditEntry, int, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}

	where := "WHERE 1=1"
	args := []any{}
	if f.Username != "" {
		where += " AND username LIKE ?"
		args = append(args, "%"+f.Username+"%")
	}
	if f.StreamPath != "" {
		where += " AND stream_path LIKE ?"
		args = append(args, "%"+f.StreamPath+"%")
	}

	var total int
	if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM audit_log %s", where), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.Limit, f.Offset)
	rows, err := db.Query(fmt.Sprintf(
		`SELECT id, username, stream_path, action, protocol, remote_addr, allowed, created_at
		 FROM audit_log %s ORDER BY created_at DESC LIMIT ? OFFSET ?`, where,
	), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []*AuditEntry
	for rows.Next() {
		var e AuditEntry
		var allowed int
		if err := rows.Scan(&e.ID, &e.Username, &e.StreamPath, &e.Action, &e.Protocol, &e.RemoteAddr, &allowed, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		e.Allowed = allowed == 1
		entries = append(entries, &e)
	}
	return entries, total, rows.Err()
}
