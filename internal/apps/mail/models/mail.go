package models

import (
	"context"
	"log/slog"
	"time"

	"github.com/uptrace/bun"
)

// TODO: Add CreatedAt, UpdatedAt time.
type Mail struct {
	ID          int64 `bun:",pk,autoincrement"`
	FromName    string
	FromAddress string `bun:",notnull"`
	Subject     string
	Text        string

	Seen      bool
	Important bool

	MailboxID int64    `bun:",notnull"`
	Mailbox   *Mailbox `bun:"rel:belongs-to,join:mailbox_id=id,on_delete:cascade"`

	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

// A method that will clean up stale emails.
func CleanupMails(ctx context.Context, db *bun.DB) error {
	slog.Info("cleaning up stale mails")

	// Delete all mails that are older than 48 hours.
	results, err := db.NewDelete().
		Model(&Mail{}).
		Where("created_at <= ?", time.Now().Add(-48*time.Hour)).
		Exec(ctx)
	if err != nil {
		slog.Debug("could not clean up stale mails", "err", err)
		return nil
	}

	rows, _ := results.RowsAffected()
	slog.Debug("cleaned up stale mails", "count", rows)
	return nil
}
