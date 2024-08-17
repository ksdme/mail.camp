package models

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/ssh"
)

// TODO: Add created at, updated at times.
type Account struct {
	ID             int64          `bun:",pk,autoincrement"`
	KeySignature   string         `bun:",notnull,unique"`
	ReservedPrefix sql.NullString `bun:",unique"`
}

func GetOrCreateAccountFromPublicKey(ctx context.Context, db *bun.DB, key ssh.PublicKey) (*Account, error) {
	hash := sha256.New()
	// key.Marshal normalizes the key to a certain extent as well.
	signature := hex.EncodeToString(hash.Sum(key.Marshal()))

	// Find existing account.
	var account Account
	err := db.
		NewSelect().
		Model(&account).
		Where("key_signature = ?", signature).
		Scan(ctx)
	if err == nil {
		return &account, nil
	} else if err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "could not query accounts")
	}

	// Create a new account.
	account = Account{KeySignature: signature}
	if _, err := db.NewInsert().Model(&account).Exec(ctx); err != nil {
		return nil, errors.Wrap(err, "could not create account")
	}

	return &account, nil
}
