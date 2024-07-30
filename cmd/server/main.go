package main

import (
	"database/sql"
	"log"
	"log/slog"

	"github.com/emersion/go-smtp"
	"github.com/ksdme/mail/internal/backend"
	"github.com/ksdme/mail/internal/config"
	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

func main() {
	if config.DevBuild {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	sqldb, err := sql.Open("sqlite3", config.DbURI)
	if err != nil {
		log.Panicf("opening db failed: %v", err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New())

	s := smtp.NewServer(backend.NewBackend(db))
	s.Addr = "127.0.0.1:1025"
	s.Domain = "localhost"
	s.AllowInsecureAuth = true

	log.Println("Starting SMTP server at", s.Addr)
	log.Fatal(s.ListenAndServe())
}
