package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/ksdme/mail/internal/config"
	"github.com/ksdme/mail/internal/models"
	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

func main() {
	sqldb, err := sql.Open("sqlite3", config.DB_URI)
	if err != nil {
		log.Panicf("opening db failed: %v", err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	db.NewCreateTable().Model(&models.Account{}).Exec(context.TODO())

	// s := smtp.NewServer(backend.NewBackend())

	// s.Addr = "127.0.0.1:1025"
	// s.Domain = "localhost"
	// s.AllowInsecureAuth = true
	// s.Debug = os.Stdout

	// log.Println("Starting SMTP server at", s.Addr)
	// log.Fatal(s.ListenAndServe())
}
