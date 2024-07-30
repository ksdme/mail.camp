package utils

import "github.com/mattn/go-sqlite3"

// Returns a boolean indicating if the current error is related to a
// database constraint failure.
func IsUniqueConstraintErr(err error) bool {
	if val, ok := err.(sqlite3.Error); ok {
		return val.ExtendedCode == sqlite3.ErrConstraintUnique
	}
	return false
}
