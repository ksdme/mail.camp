package mail

import (
	"io"
	"log"
	"net/mail"

	"github.com/emersion/go-smtp"
)

func NewBackend() *backend {
	return &backend{}
}

// The SMTP server backend. At the moment, it does not support
// outgoing messages.
type backend struct{}

func (bkd *backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &session{}, nil
}

// A session on the backend.
type session struct {
	from    string
	subject string
	content string
}

// Handles the MAIL command. It is typically used to indicate whether
// the sender address is accepted on this server. The upstream MTA
// will use it to bounce route the email.
// TODO: Check a blacklist?
func (s *session) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	return nil
}

// Handles the RCPT command. Each instance of this command specifies a
// recipient email address. It is typically also useful to indicate
// whether a recipient address is accepted.
func (s *session) Rcpt(to string, opts *smtp.RcptOptions) error {
	// TODO: Check if the recipient email address is known.
	return nil
}

// Handles the DATA command. It will be called to receive the email contents,
// including the headers, subject, body and inline or file attachments.
func (s *session) Data(r io.Reader) error {
	message, _ := mail.ReadMessage(r)

	s.subject = message.Header.Get("Subject")
	if content, err := extractPlainText(message); err != nil {
		log.Printf("could not extract text from email: %v", err)
	} else {
		s.content = content
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
	s.from = ""
	s.subject = ""
	s.content = ""
}
