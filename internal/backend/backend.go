package backend

import (
	"context"
	"io"
	"log/slog"
	"net/mail"

	"github.com/emersion/go-smtp"
	"github.com/ksdme/mail/internal/models"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

func NewBackend(db *bun.DB) *backend {
	return &backend{db}
}

// The SMTP server backend. At the moment, it does not support
// outgoing messages.
type backend struct {
	db *bun.DB
}

func (b *backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &session{db: b.db}, nil
}

// A session on the backend.
type session struct {
	db        *bun.DB
	mailboxes []models.Mailbox
}

// Handles the MAIL command. It is typically used to indicate whether
// the sender address is accepted on this server. The upstream MTA
// will use it to bounce route the email.
func (s *session) Mail(from string, opts *smtp.MailOptions) error {
	// TODO: Check a blacklist?
	slog.Debug("> MAIL", "from", from)
	return nil
}

// Handles the RCPT command. Each instance of this command specifies a
// recipient email address. It is typically also useful to indicate
// whether a recipient address is accepted.
func (s *session) Rcpt(to string, opts *smtp.RcptOptions) error {
	// TODO: Check if the recipient email address is known.
	// TODO: Check if they hit a limit, maybe a mailbox count limit?
	slog.Debug("> RCPT", "to", to)
	return nil
}

// Handles the DATA command. It will be called to receive the email contents,
// including the headers, subject, body and inline or file attachments.
func (s *session) Data(r io.Reader) error {
	message, _ := mail.ReadMessage(r)

	text, err := extractPlainText(message)
	if err != nil {
		return errors.Wrap(err, "could not read message")
	}

	from := message.Header.Get("From")
	subject := message.Header.Get("Subject")
	for _, mailbox := range s.mailboxes {
		mail := &models.Mail{
			From:      from,
			Subject:   subject,
			Text:      text,
			MailboxID: mailbox.ID,
		}

		_, err := s.db.NewInsert().Model(mail).Exec(context.Background())
		if err != nil {
			slog.Info("could not add mail to mailbox", "from", from, "mailbox", mailbox.ID, "err", err)
		} else {
			slog.Debug("added mail to mailbox", "from", from, "mailbox", mailbox.ID)
		}
	}

	return nil
}

// Perform clean up on this session.
func (s *session) Logout() error {
	return nil
}

// Handles the RSET command. It is typically useful for aborting the current
// mail transaction. This allows the sender to reuse the connection for sending
// another email.
func (s *session) Reset() {
	var mailboxes []models.Mailbox
	s.mailboxes = mailboxes
}
