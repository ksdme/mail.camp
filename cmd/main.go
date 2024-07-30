package main

import (
	"log"
	"os"

	"github.com/emersion/go-smtp"
	"github.com/ksdme/mail/internal/backend"
)

func main() {
	s := smtp.NewServer(backend.NewBackend())

	s.Addr = "127.0.0.1:1025"
	s.Domain = "localhost"
	s.AllowInsecureAuth = true
	s.Debug = os.Stdout

	log.Println("Starting SMTP server at", s.Addr)
	log.Fatal(s.ListenAndServe())
}
