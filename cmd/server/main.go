package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/ksdme/mail/internal/backend"
	"github.com/ksdme/mail/internal/config"
	"github.com/ksdme/mail/internal/models"
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

	// SMTP Server
	s := smtp.NewServer(backend.NewBackend(db))
	s.Addr = "127.0.0.1:1025"
	s.Domain = "localhost"
	s.AllowInsecureAuth = true
	go runMailServer(s)

	// SSH Server

	// Background Workers
	go runCleanupWorker(db)
}

func runMailServer(s *smtp.Server) {
	slog.Info("starting smtp server", "at", s.Addr)
	slog.Error("failed serving smtp server", "err", s.ListenAndServe())
}

func runCleanupWorker(db *bun.DB) {
	for {
		models.CleanupMails(context.Background(), db)
		time.Sleep(time.Hour)
	}
}
