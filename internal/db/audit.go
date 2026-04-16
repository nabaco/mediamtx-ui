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
