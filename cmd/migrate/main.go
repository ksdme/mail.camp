package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"

	"github.com/ksdme/mail/internal/config"
	"github.com/ksdme/mail/internal/models"
	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/extra/bundebug"
)

func main() {
	sqldb, err := sql.Open("sqlite3", config.DbURI)
	if err != nil {
		log.Panicf("opening db failed: %v", err)
	}

	ctx := context.Background()
	db := bun.NewDB(sqldb, sqlitedialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	// TODO: This will create tables with the latest schema. This will not work if
	// the database is already on an older version of the schema. We need to actually
	// support some sort of incremental migration.
	// https://bun.uptrace.dev/guide/migrations.html
	must(db.NewCreateTable().Model(&models.Account{}).Exec(ctx))
	must(db.NewCreateTable().Model(&models.Mailbox{}).Exec(ctx))
	must(db.NewCreateTable().Model(&models.Mail{}).Exec(ctx))
	slog.Info("created tables")

	if config.DevBuild {
		account := &models.Account{KeySignature: "dev-signature"}
		must(db.NewInsert().Model(account).Exec(ctx))
		slog.Info("created account", "id", account.ID)

		mailbox := &models.Mailbox{Name: "dev-mailbox", AccountID: account.ID}
		must(db.NewInsert().Model(mailbox).Exec(ctx))
		slog.Info("created mailbox", "id", mailbox.ID)
	}
}

func must(result sql.Result, err error) sql.Result {
	if err != nil {
		log.Panicf("could not run query: %v", err)
	}
	return result
}
