package backend

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/mail"
	"strings"

	"github.com/emersion/go-smtp"
	"github.com/ksdme/mail/internal/config"
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
	from      *mail.Address
	mailboxes []models.Mailbox
}

// Handles the MAIL command. It is typically used to indicate whether
// the sender address is accepted on this server. The upstream MTA
// will use it to bounce route the email.
func (s *session) Mail(from string, opts *smtp.MailOptions) error {
	// TODO: Check a blacklist?
	slog.Debug("> MAIL", "from", from)

	address, err := mail.ParseAddress(from)
	if err != nil {
		return errors.Wrap(err, "could not parse from address")
	}
	s.from = address

	return nil
}

// Handles the RCPT command. Each instance of this command specifies a
// recipient email address. It is typically also useful to indicate
// whether a recipient address is accepted.
func (s *session) Rcpt(to string, opts *smtp.RcptOptions) error {
	// TODO: Check if the recipient email address is known.
	// TODO: Check if they hit a limit, maybe a mailbox count limit?
	slog.Debug("> RCPT", "to", to)

	// Parse and validate the email address at the same time.
	recipient, err := mail.ParseAddress(to)
	if err != nil {
		return errors.Wrap(err, "could not parse recipient address")
	}

	domain := fmt.Sprintf("@%s", config.MX_DOMAIN)
	if !strings.HasSuffix(recipient.Address, domain) {
		return fmt.Errorf("unrecognized domain: %v", recipient.Address)
	}

	// The name of the target mailbox.
	name := strings.Split(recipient.Address, "@")[0]
	found := false
	mailbox := models.Mailbox{}

	// Check if such a mailbox already exists.
	err = s.db.NewSelect().Model(&mailbox).Where("name = ?", name).Scan(context.Background())
	if err == nil {
		found = true
	} else if err != sql.ErrNoRows {
		return errors.Wrap(err, "querying for mailboxes failed")
	}

	// TODO: Create a mailbox if necessary.

	if found {
		slog.Debug("found matching mailbox", "mailbox", mailbox.ID)
		s.mailboxes = append(s.mailboxes, mailbox)
		return nil
	}

	return fmt.Errorf("could not find a mailbox for %s", to)
}

// Handles the DATA command. It will be called to receive the email contents,
// including the headers, subject, body and inline or file attachments.
func (s *session) Data(r io.Reader) error {
	message, _ := mail.ReadMessage(r)

	text, err := extractPlainText(message)
	if err != nil {
		return errors.Wrap(err, "could not read message")
	}

	for _, mailbox := range s.mailboxes {
		mail := &models.Mail{
			FromAddress: s.from.Address,
			FromName:    s.from.Name,
			Subject:     message.Header.Get("Subject"),
			Text:        text,
			MailboxID:   mailbox.ID,
		}

		_, err := s.db.NewInsert().Model(mail).Exec(context.Background())
		if err != nil {
			slog.Info("could not add mail to mailbox", "from", s.from.Address, "mailbox", mailbox.ID, "err", err)
		} else {
			slog.Debug("added mail to mailbox", "from", s.from.Address, "mailbox", mailbox.ID)
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
	s.from = nil
}
