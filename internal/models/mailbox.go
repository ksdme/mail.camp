package models

// TODO: Add CreatedAt, UpdatedAt fields.
type Mailbox struct {
	ID   int64  `bun:",pk,autoincrement"`
	Name string `bun:",notnull"`

	AccountID int64    `bun:",notnull"`
	Account   *Account `bun:"rel:belongs-to,join:account_id=id"`
}

// Add an email to the mailbox.
// TODO: Add
func (m *Mailbox) Add(mail *Mail) error {
	return nil
}
