package models

import "time"

// TODO: Add CreatedAt, UpdatedAt time.
type Mail struct {
	ID          int64 `bun:",pk,autoincrement"`
	FromName    string
	FromAddress string `bun:",notnull"`
	Subject     string
	Text        string

	Seen      bool
	Important bool

	// TODO: We need to setup cascade relationship.
	MailboxID int64    `bun:",notnull"`
	Mailbox   *Mailbox `bun:"rel:belongs-to,join:mailbox_id=id"`

	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}
