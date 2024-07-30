package models

// TODO: Add CreatedAt, UpdatedAt time.
type Mail struct {
	ID          int64 `bun:",pk,autoincrement"`
	FromName    string
	FromAddress string `bun:",notnull"`
	Subject     string
	Text        string

	Important bool

	MailboxID int64    `bun:",notnull"`
	Mailbox   *Mailbox `bun:"rel:belongs-to,join:mailbox_id=id"`
}
