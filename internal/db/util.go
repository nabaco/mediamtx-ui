package db

import "strings"

// isUniqueViolation returns true when err is a SQLite UNIQUE constraint violation.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
