package main

import (
	"context"
	"database/sql"
	"io"
	"log"
	"log/slog"
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/emersion/go-smtp"
	"github.com/ksdme/mail/internal/backend"
	"github.com/ksdme/mail/internal/config"
	"github.com/ksdme/mail/internal/models"
	"github.com/ksdme/mail/internal/tui"
	"github.com/ksdme/mail/internal/tui/colors"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
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

	// Start servers and workers.
	var wg sync.WaitGroup

	wg.Add(1)
	go runSSHServer(db, &wg)

	wg.Add(1)
	go runMailServer(db, &wg)

	wg.Add(1)
	go runCleanupWorker(db, &wg)

	wg.Wait()
}

func runMailServer(db *bun.DB, wg *sync.WaitGroup) {
	defer wg.Done()

	server := smtp.NewServer(backend.NewBackend(db))
	server.Addr = config.SMTPBindAddr
	server.Domain = config.MxHost

	slog.Info("starting smtp server", "at", config.SMTPBindAddr)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("failed serving smtp server", "err", err)
	}
}

func runSSHServer(db *bun.DB, wg *sync.WaitGroup) {
	defer wg.Done()

	options := []ssh.Option{
		wish.WithAddress(config.SSHBindAddr),
		wish.WithHostKeyPath(config.SSHHostKeyPath),
	}

	if config.SSHAuthorizedKeysPath == "" {
		options = append(options, wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			return true
		}))
	} else {
		if _, err := os.Stat(config.SSHAuthorizedKeysPath); err != nil {
			panic(errors.Wrap(err, "could not stat authorized_keys file"))
		}
		options = append(options, wish.WithAuthorizedKeys(config.SSHAuthorizedKeysPath))
	}

	options = append(options, wish.WithMiddleware(
		bubbletea.Middleware(func(session ssh.Session) (tea.Model, []tea.ProgramOption) {
			account, err := models.GetOrCreateAccountFromPublicKey(
				session.Context(),
				db,
				session.PublicKey(),
			)
			if err != nil {
				io.WriteString(session, err.Error()+"\n")
				return nil, nil
			}

			// TODO: We should adjust the color palette based on term color.
			model := tui.NewModel(db, *account, colors.DefaultColorPalette())
			options := []tea.ProgramOption{tea.WithAltScreen()}
			return model, options
		}),
	))

	server, err := wish.NewServer(options...)
	if err != nil {
		slog.Error("failed creating ssh server", "err", err)
	}

	slog.Info("starting ssh server", "at", config.SSHBindAddr)
	if err = server.ListenAndServe(); err != nil {
		slog.Error("failed serving ssh connections", "err", err)
	}
}

func runCleanupWorker(db *bun.DB, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		models.CleanupMails(context.Background(), db)
		time.Sleep(time.Hour)
	}
}
