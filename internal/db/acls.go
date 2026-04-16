package db

import (
	"database/sql"
	"fmt"
	"path"
	"time"
)

type ACLAction string

const (
	ACLActionRead    ACLAction = "read"
	ACLActionPublish ACLAction = "publish"
	ACLActionBoth    ACLAction = "both"
)

type SubjectType string

const (
	SubjectUser  SubjectType = "user"
	SubjectGroup SubjectType = "group"
)

type ACL struct {
	ID            int64
	SubjectType   SubjectType
	SubjectID     int64
	SubjectName   string // populated from join
	StreamPattern string
	Action        ACLAction
	CreatedAt     time.Time
}

func CreateACL(db *sql.DB, subjectType SubjectType, subjectID int64, streamPattern string, action ACLAction) (*ACL, error) {
	res, err := db.Exec(
		`INSERT INTO acls (subject_type, subject_id, stream_pattern, action) VALUES (?, ?, ?, ?)`,
		string(subjectType), subjectID, streamPattern, string(action),
	)
	if err != nil {
		return nil, fmt.Errorf("create acl: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetACLByID(db, id)
}

func GetACLByID(db *sql.DB, id int64) (*ACL, error) {
	var a ACL
	err := db.QueryRow(
		`SELECT id, subject_type, subject_id, stream_pattern, action, created_at FROM acls WHERE id = ?`, id,
	).Scan(&a.ID, &a.SubjectType, &a.SubjectID, &a.StreamPattern, &a.Action, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &a, err
}

func ListACLs(db *sql.DB) ([]*ACL, error) {
	rows, err := db.Query(`
		SELECT a.id, a.subject_type, a.subject_id,
		       COALESCE(u.username, g.name, '') AS subject_name,
		       a.stream_pattern, a.action, a.created_at
		FROM acls a
		LEFT JOIN users  u ON a.subject_type = 'user'  AND a.subject_id = u.id
		LEFT JOIN groups g ON a.subject_type = 'group' AND a.subject_id = g.id
		ORDER BY a.subject_type, subject_name, a.stream_pattern
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var acls []*ACL
	for rows.Next() {
		var a ACL
		if err := rows.Scan(&a.ID, &a.SubjectType, &a.SubjectID, &a.SubjectName, &a.StreamPattern, &a.Action, &a.CreatedAt); err != nil {
			return nil, err
		}
		acls = append(acls, &a)
	}
	return acls, rows.Err()
}

func DeleteACL(db *sql.DB, id int64) error {
	res, err := db.Exec(`DELETE FROM acls WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete acl: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// CheckAccess returns true if the user (by ID) is allowed to perform action on streamPath.
// It checks direct user ACLs and all group ACLs the user belongs to.
func CheckAccess(db *sql.DB, userID int64, streamPath string, action ACLAction) (bool, error) {
	groupIDs, err := GetUserGroupIDs(db, userID)
	if err != nil {
		return false, err
	}

	// Gather all ACLs applicable to this user (direct + group)
	rows, err := db.Query(`
		SELECT stream_pattern, action FROM acls
		WHERE (subject_type = 'user' AND subject_id = ?)
		   OR (subject_type = 'group' AND subject_id IN (`+placeholders(len(groupIDs))+`))
	`, append([]any{userID}, int64SliceToAny(groupIDs)...)...)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var pattern string
		var aclAction ACLAction
		if err := rows.Scan(&pattern, &aclAction); err != nil {
			return false, err
		}
		if matchesAction(aclAction, action) {
			if ok, _ := path.Match(pattern, streamPath); ok {
				return true, nil
			}
		}
	}
	return false, rows.Err()
}

func matchesAction(aclAction, requested ACLAction) bool {
	return aclAction == ACLActionBoth || aclAction == requested
}

func placeholders(n int) string {
	if n == 0 {
		return "NULL"
	}
	s := "?"
	for i := 1; i < n; i++ {
		s += ",?"
	}
	return s
}

func int64SliceToAny(ids []int64) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}
