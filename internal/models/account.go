package models

import (
	"database/sql"
)

// TODO: Add created at, updated at times.
type Account struct {
	ID             int64          `bun:",pk,autoincrement"`
	KeySignature   string         `bun:",notnull,unique"`
	ReservedPrefix sql.NullString `bun:",unique"`
}
