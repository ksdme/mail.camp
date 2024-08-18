package models

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"io"
	"log/slog"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/ksdme/mail/internal/config"
	core "github.com/ksdme/mail/internal/core/models"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

// A shared clipboard item. The actual data on the item is encrypted using
// AES-128 with the normalized public key of the user and the server entropy
// as the encryption key. This is not meant to be a fool proof encryption
// scheme. Instead, the idea is to prevent leaking any sensitive information
// readily given that clipboards might contain sensitive information.
// TODO: We can make this completely in-memory instead of writing to a table.
type ClipboardItem struct {
	ID int64 `bun:",pk,autoincrement"`

	Value []byte `bun:",notnull"`

	// TODO: We need to setup cascade relationship.
	AccountID int64         `bun:",notnull,unique"`
	Account   *core.Account `bun:"rel:belongs-to,join:account_id=id"`

	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

// Create an encrypted clipboard item from a value.
func CreateClipboardItem(ctx context.Context, db *bun.DB, value []byte, key ssh.PublicKey, account core.Account) error {
	slog.Debug("creating clipboard item", "account", account.ID)

	// TODO: Validate size and existence.

	// Remove any existing clipboard items on this account.
	_, err := db.NewDelete().
		Model(&ClipboardItem{}).
		Where("account_id = ?", account.ID).
		Exec(ctx)
	if err != nil {
		return errors.Wrap(err, "could not empty clipboard items")
	}

	// Insert the new item.
	// Prepare none and add it to the input.
	nonce := make([]byte, 64)
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return errors.Wrap(err, "could not generate iv")
	}
	value = append(value, nonce...)

	// Generate the key and cipher.
	cipher, err := makeCipher(key)
	if err != nil {
		return err
	}

	// Encrypt the value.
	encrypted := make([]byte, len(value))
	cipher.Encrypt(encrypted, value)

	// Write the value.
	item := ClipboardItem{
		Value:     encrypted,
		AccountID: account.ID,
	}
	if _, err = db.NewInsert().Model(&item).Exec(ctx); err != nil {
		return errors.Wrap(err, "could not write to database")
	}

	return nil
}

// Returns a decrypted clipboard item if it exists, otherwise, nil.
func GetClipboardValue(ctx context.Context, db *bun.DB, key ssh.PublicKey, account core.Account) ([]byte, error) {
	// Find the item.
	var item ClipboardItem
	err := db.
		NewSelect().
		Model(&item).
		Where("account_id = ?", account.ID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, errors.Wrap(err, "could not query clipboard items")
	}

	// Decrypt the value.
	cipher, err := makeCipher(key)
	if err != nil {
		return nil, nil
	}
	decrypted := make([]byte, len(item.Value))
	cipher.Decrypt(decrypted, item.Value)

	// Remove the nonce.
	return decrypted[:len(decrypted)-64], nil
}

func makeCipher(key ssh.PublicKey) (cipher.Block, error) {
	basis := append(key.Marshal(), []byte(config.Core.Entropy)...)

	// TODO: Use a better mode with actual IV.
	cipher, err := aes.NewCipher(sha256.New().Sum(basis))
	if err != nil {
		return nil, errors.Wrap(err, "could not generate cipher")
	}

	return cipher, nil
}
