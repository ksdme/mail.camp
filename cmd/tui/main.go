package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ksdme/mail/internal/config"
	"github.com/ksdme/mail/internal/models"
	"github.com/ksdme/mail/internal/tui"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	sqldb, err := sql.Open("sqlite3", config.DbURI)
	if err != nil {
		log.Panicf("opening db failed: %v", err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New())

	var account models.Account
	if err := db.NewSelect().Model(&account).Scan(context.Background()); err != nil {
		panic(err)
	}
	slog.Debug("Using", "account", account)

	program := tea.NewProgram(tui.NewModel(db, account))
	if _, err := program.Run(); err != nil {
		os.Exit(1)
	}
}
