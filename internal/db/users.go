package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type User struct {
	ID              int64
	Username        string
	PasswordHash    string
	StreamTokenHash *string
	Role            Role
	Enabled         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("already exists")

func CreateUser(db *sql.DB, username, passwordHash string, role Role) (*User, error) {
	res, err := db.Exec(
		`INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`,
		username, passwordHash, string(role),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetUserByID(db, id)
}

func GetUserByID(db *sql.DB, id int64) (*User, error) {
	return scanUser(db.QueryRow(
		`SELECT id, username, password_hash, stream_token_hash, role, enabled, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	))
}

func GetUserByUsername(db *sql.DB, username string) (*User, error) {
	return scanUser(db.QueryRow(
		`SELECT id, username, password_hash, stream_token_hash, role, enabled, created_at, updated_at
		 FROM users WHERE username = ?`, username,
	))
}

func ListUsers(db *sql.DB) ([]*User, error) {
	rows, err := db.Query(
		`SELECT id, username, password_hash, stream_token_hash, role, enabled, created_at, updated_at
		 FROM users ORDER BY username`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

type UpdateUserParams struct {
	PasswordHash    *string
	StreamTokenHash *string // nil = no change, "" = clear
	Role            *Role
	Enabled         *bool
}

func UpdateUser(db *sql.DB, id int64, p UpdateUserParams) (*User, error) {
	u, err := GetUserByID(db, id)
	if err != nil {
		return nil, err
	}

	if p.PasswordHash != nil {
		u.PasswordHash = *p.PasswordHash
	}
	if p.StreamTokenHash != nil {
		u.StreamTokenHash = p.StreamTokenHash
	}
	if p.Role != nil {
		u.Role = *p.Role
	}
	if p.Enabled != nil {
		u.Enabled = *p.Enabled
	}

	enabled := 0
	if u.Enabled {
		enabled = 1
	}

	_, err = db.Exec(
		`UPDATE users SET password_hash=?, stream_token_hash=?, role=?, enabled=?, updated_at=CURRENT_TIMESTAMP
		 WHERE id=?`,
		u.PasswordHash, u.StreamTokenHash, string(u.Role), enabled, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return GetUserByID(db, id)
}

func DeleteUser(db *sql.DB, id int64) error {
	res, err := db.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func CountUsers(db *sql.DB) (int, error) {
	var n int
	return n, db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
}

func scanUser(row *sql.Row) (*User, error) {
	var u User
	var enabled int
	var sth sql.NullString
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &sth, &u.Role, &enabled, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.Enabled = enabled == 1
	if sth.Valid {
		u.StreamTokenHash = &sth.String
	}
	return &u, nil
}

func scanUserRow(row *sql.Rows) (*User, error) {
	var u User
	var enabled int
	var sth sql.NullString
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &sth, &u.Role, &enabled, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.Enabled = enabled == 1
	if sth.Valid {
		u.StreamTokenHash = &sth.String
	}
	return &u, nil
}
