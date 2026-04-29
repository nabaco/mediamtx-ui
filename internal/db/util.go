package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"
)

// isUniqueViolation returns true when err is a SQLite UNIQUE constraint violation.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// PublishSlug returns an 8-hex-char slug derived from a stream token.
// Used for credential-less RTMP publish via ?token={slug}.
// Rotates automatically when the token is regenerated.
func PublishSlug(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])[:8]
}

// GetUserByPublishSlug finds an enabled user whose stream token derives to the given slug.
func GetUserByPublishSlug(db *sql.DB, slug string) (*User, error) {
	users, err := ListUsers(db)
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		if !u.Enabled || u.StreamTokenHash == nil {
			continue
		}
		if PublishSlug(*u.StreamTokenHash) == slug {
			return u, nil
		}
	}
	return nil, ErrNotFound
}
