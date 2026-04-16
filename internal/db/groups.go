package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Group struct {
	ID        int64
	Name      string
	CreatedAt time.Time
	Members   []*User // populated optionally
}

func CreateGroup(db *sql.DB, name string) (*Group, error) {
	res, err := db.Exec(`INSERT INTO groups (name) VALUES (?)`, name)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("create group: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetGroupByID(db, id)
}

func GetGroupByID(db *sql.DB, id int64) (*Group, error) {
	var g Group
	err := db.QueryRow(`SELECT id, name, created_at FROM groups WHERE id = ?`, id).
		Scan(&g.ID, &g.Name, &g.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func ListGroups(db *sql.DB) ([]*Group, error) {
	rows, err := db.Query(`SELECT id, name, created_at FROM groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, &g)
	}
	return groups, rows.Err()
}

func RenameGroup(db *sql.DB, id int64, name string) (*Group, error) {
	res, err := db.Exec(`UPDATE groups SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("rename group: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return GetGroupByID(db, id)
}

func DeleteGroup(db *sql.DB, id int64) error {
	res, err := db.Exec(`DELETE FROM groups WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func AddGroupMember(db *sql.DB, groupID, userID int64) error {
	_, err := db.Exec(`INSERT OR IGNORE INTO user_groups (user_id, group_id) VALUES (?, ?)`, userID, groupID)
	return err
}

func RemoveGroupMember(db *sql.DB, groupID, userID int64) error {
	_, err := db.Exec(`DELETE FROM user_groups WHERE user_id = ? AND group_id = ?`, userID, groupID)
	return err
}

func GetGroupMembers(db *sql.DB, groupID int64) ([]*User, error) {
	rows, err := db.Query(
		`SELECT u.id, u.username, u.password_hash, u.stream_token_hash, u.role, u.enabled, u.created_at, u.updated_at
		 FROM users u
		 JOIN user_groups ug ON ug.user_id = u.id
		 WHERE ug.group_id = ?
		 ORDER BY u.username`, groupID,
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

// GetUserGroupIDs returns all group IDs a user belongs to.
func GetUserGroupIDs(db *sql.DB, userID int64) ([]int64, error) {
	rows, err := db.Query(`SELECT group_id FROM user_groups WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
