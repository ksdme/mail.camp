package models

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
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

// A weak Base64 validation pattern.
var fingerprintValuePattern = regexp.MustCompile(`^[a-zA-Z0-9\+\/\=]+$`)

// Add a key to the account.
func (account *Account) AddKey(ctx context.Context, db *bun.DB, fingerprint string) error {
	fingerprint = normalize(fingerprint)

	// Validate the key.
	partials := strings.Split(fingerprint, ":")
	if len(partials) != 2 {
		return fmt.Errorf("bad fingerprint: use Base64 encoded SHA256")
	}
	// Eh, the only reason we prevent other algorithms is because in go-land,
	// we can only generate SHA256 fingerprints of incoming public keys.
	if partials[0] != "SHA256" {
		return fmt.Errorf("bad fingerprint: unsupported hash algorithm, use SHA256")
	}
	if !fingerprintValuePattern.MatchString(partials[1]) {
		return fmt.Errorf("bad fingerprint: unknown encoding, use Base64 encoded SHA256")
	}

	err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		key := Key{}

		// Check if the key is taken.
		err := tx.NewSelect().Model(&key).Where("fingerprint = ?", fingerprint).Scan(ctx)
		if err == nil {
			if key.AccountID == account.ID {
				return fmt.Errorf("this key was already added to your account")
			}
			return errors.Wrap(err, "this key is already attached to another account")
		}
		if err != sql.ErrNoRows {
			return errors.Wrap(err, "could not query keys")
		}

		// Add the key otherwise.
		key = Key{Fingerprint: fingerprint, AccountID: account.ID}
		if _, err := tx.NewInsert().Model(&key).Exec(ctx); err != nil {
			return errors.Wrap(err, "could not create key")
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// Remove a key from a specific account.
func (a *Account) RemoveKey(
	ctx context.Context,
	db *bun.DB,
	fingerprint string,
) error {
	fingerprint = normalize(fingerprint)
	slog.Info("deleting key", "account", a.ID, "fingerprint", fingerprint)

	// Delete the key if possible.
	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		result, err := tx.
			NewDelete().
			Model(&Key{}).
			Where("account_id = ?", a.ID).
			Where("fingerprint = ?", fingerprint).
			Exec(ctx)
		if err != nil {
			return errors.Wrap(err, "could not query")
		}
		if count, err := result.RowsAffected(); err != nil {
			return errors.Wrap(err, "could not query")
		} else {
			if count == 0 {
				return fmt.Errorf("key not found")
			}
		}

		count, err := tx.
			NewSelect().
			Model(&Key{}).
			Where("account_id = ?", a.ID).
			Count(ctx)
		if err != nil {
			return errors.Wrap(err, "could not query keys")
		}
		if count == 0 {
			return fmt.Errorf("this will leave your account without any keys")
		}

		return nil
	})
}

// List all the keys added on this account.
func (a *Account) ListKeys(ctx context.Context, db *bun.DB) ([]Key, error) {
	var keys []Key

	err := db.
		NewSelect().
		Model(&keys).
		Where("account_id = ?", a.ID).
		Order("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not query keys")
	}

	return keys, nil
}

// Retrieve or create an account.
func GetOrCreateAccountFromPublicKey(
	ctx context.Context,
	db *bun.DB,
	key ssh.PublicKey,
) (*Account, error) {
	fingerprint := ssh.FingerprintSHA256(key)

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
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		account = Account{}
		if _, err := tx.NewInsert().Model(&account).Exec(ctx); err != nil {
			return errors.Wrap(err, "could not create account")
		}

		// TODO: This should just use account.AddKey.
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

func normalize(fingerprint string) string {
	return strings.TrimSpace(fingerprint)
}
