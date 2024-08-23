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
	"github.com/ksdme/mail/internal/apps/clipboard/events"
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

	IV    []byte `bun:",notnull"`
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

	// Clear out the existing clipboard.
	if err := DeleteClipboard(ctx, db, account); err != nil {
		return err
	}

	// Generate the cipher.
	aes, err := makeAESCipher(key)
	if err != nil {
		return err
	}

	// Generate a random IV.
	iv := make([]byte, aes.BlockSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return errors.Wrap(err, "could not generate iv")
	}

	// Encrypt.
	ciphered := make([]byte, len(value))
	cfb := cipher.NewCFBEncrypter(aes, iv)
	cfb.XORKeyStream(ciphered, value)

	// Write the value.
	item := ClipboardItem{
		IV:        iv,
		Value:     ciphered,
		AccountID: account.ID,
	}
	if _, err = db.NewInsert().Model(&item).Exec(ctx); err != nil {
		return errors.Wrap(err, "could not write to database")
	}

	// If there is an interactive session somewhere, trigger an update there.
	events.ClipboardContentsUpdatedSignal.Emit(account.ID, item.ID)

	return nil
}

type DecodedClipboardItem struct {
	Value     []byte
	CreatedAt time.Time
}

// Returns a decrypted clipboard item if it exists, otherwise, nil.
func GetClipboardValue(ctx context.Context, db *bun.DB, key ssh.PublicKey, account core.Account) (*DecodedClipboardItem, error) {
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

	// Generate cipher.
	aes, err := makeAESCipher(key)
	if err != nil {
		return nil, nil
	}

	// Decipher.
	deciphered := make([]byte, len(item.Value))
	cfb := cipher.NewCFBDecrypter(aes, item.IV)
	cfb.XORKeyStream(deciphered, item.Value)

	return &DecodedClipboardItem{
		Value:     deciphered,
		CreatedAt: item.CreatedAt,
	}, nil
}

// Remove existing clipboard items on this account.
func DeleteClipboard(ctx context.Context, db *bun.DB, account core.Account) error {
	_, err := db.NewDelete().
		Model(&ClipboardItem{}).
		Where("account_id = ?", account.ID).
		Exec(ctx)
	if err != nil {
		return errors.Wrap(err, "could not empty clipboard items")
	}
	return nil
}

// Remove all the clipboard items.
func CleanAll(ctx context.Context, db *bun.DB) error {
	_, err := db.NewDelete().Model(&ClipboardItem{}).Exec(ctx)
	slog.Info("cleaning up all clipboard items")
	return err
}

// Remove all the expired clipboard items.
func CleanUp(ctx context.Context, db *bun.DB) error {
	_, err := db.
		NewDelete().
		Model(&ClipboardItem{}).
		Where("created_at <= ?", time.Now().Add(5*time.Minute)).
		Exec(ctx)
	slog.Info("cleaning up expired clipboard items")
	return err
}

func makeAESCipher(key ssh.PublicKey) (cipher.Block, error) {
	basis := append(key.Marshal(), []byte(config.Core.Entropy)...)
	hash := sha256.New()
	if _, err := hash.Write(basis); err != nil {
		return nil, errors.Wrap(err, "could not hash key")
	}

	cipher, err := aes.NewCipher(hash.Sum(nil))
	if err != nil {
		return nil, errors.Wrap(err, "could not generate cipher")
	}

	return cipher, nil
}
