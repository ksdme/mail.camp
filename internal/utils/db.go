package utils

import (
	"database/sql"
	"log"

	"github.com/mattn/go-sqlite3"
)

// Returns a boolean indicating if the current error is related to a
// database constraint failure.
func IsUniqueConstraintErr(err error) bool {
	if val, ok := err.(sqlite3.Error); ok {
		return val.ExtendedCode == sqlite3.ErrConstraintUnique
	}
	return false
}

func MustExec(result sql.Result, err error) sql.Result {
	if err != nil {
		log.Panicf("could not run query: %v", err)
	}
	return result
}
