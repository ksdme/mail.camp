package main

import (
	"fmt"
	"io"
	"mime/multipart"
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
type session struct{}

// Handles the MAIL command. It is typically used to indicate whether
// the sender address is accepted on this server. The upstream MTA
// will use it to bounce route the email.
func (s *session) Mail(from string, opts *smtp.MailOptions) error {
	// TODO: Check a blacklist?
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
	fmt.Println("----- Data")
	e, _ := mail.ReadMessage(r)
	fmt.Println(">>>", e.Header.Get("Content-Type"))

	// mime.ParseMediaType()
	m := multipart.NewReader(r, "")
	m.NextPart()

	body, _ := io.ReadAll(e.Body)
	fmt.Println("> Body", string(body))
	return nil
}

// Perform clean up on this session.
func (s *session) Logout() error {
	return nil
}

// Handles the RSET command. It is typically useful for aborting the current
// mail transaction. This allows the sender to reuse the connection for sending
// another email.
func (s *session) Reset() {}
