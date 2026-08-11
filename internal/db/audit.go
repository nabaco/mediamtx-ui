package db

import (
	"database/sql"
	"fmt"
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
func VacuumDB(db *sql.DB) error {
	if _, err := db.Exec("VACUUM"); err != nil {
		return fmt.Errorf("vacuum: %w", err)
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
