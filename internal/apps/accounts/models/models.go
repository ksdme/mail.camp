package models

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ksdme/mail/internal/utils"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/ssh"
)

// Represents an account on the platform.
type Account struct {
	ID             int64          `bun:",pk,autoincrement"`
	ReservedPrefix sql.NullString `bun:",unique"`
	CreatedAt      time.Time      `bun:",nullzero,notnull,default:current_timestamp"`
}

// A login key associated with the account.
type Key struct {
	ID          int64  `bun:",pk,autoincrement"`
	Fingerprint string `bun:",notnull,unique"`

	// TODO: We need to setup cascade relationship.
	AccountID int64    `bun:",notnull"`
	Account   *Account `bun:"rel:belongs-to,join:account_id=id"`

	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

// Retrieve or create an account.
func GetOrCreateAccountFromPublicKey(
	ctx context.Context,
	db *bun.DB,
	key ssh.PublicKey,
) (*Account, error) {
	fingerprint := ssh.FingerprintSHA256(key)
	fingerprint = normalize(fingerprint)

	// Find existing account.
	var account Account
	err := db.
		NewSelect().
		Model(&account).
		Join("JOIN keys as key").
		JoinOn("key.account_id = account.id").
		Where("key.fingerprint = ?", fingerprint).
		Scan(ctx)
	if err == nil {
		return &account, nil
	} else if err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "could not query accounts")
	}

	// Create an account if one doesn't exist.
	err = db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		account = Account{}
		if _, err := tx.NewInsert().Model(&account).Exec(ctx); err != nil {
			return errors.Wrap(err, "could not create account")
		}

		login := Key{Fingerprint: fingerprint, AccountID: account.ID}
		if _, err := tx.NewInsert().Model(&login).Exec(ctx); err != nil {
			if utils.IsUniqueConstraintErr(err) {
				return fmt.Errorf("an account with this key already exists")
			}
			return errors.Wrap(err, "could not create account")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &account, nil
}

// Remove a key from a specific account.
func DeleteKey(
	ctx context.Context,
	db *bun.DB,
	account Account,
	fingerprint string,
) error {
	fingerprint = normalize(fingerprint)
	slog.Info("deleting account key", "account", account.ID, "fingerprint", fingerprint)

	_, err := db.
		NewDelete().
		Model(&Key{}).
		Where("account_id = ?", account.ID).
		Where("fingerprint = ?", fingerprint).
		Exec(ctx)
	if err != nil {
		return errors.Wrap(err, "could not delete key")
	}

	return nil
}

func normalize(fingerprint string) string {
	// While the display format of ssh-keygen has both lowercase and uppercase
	// characters, the hash is case insensitive.
	return strings.ToLower(fingerprint)
}
