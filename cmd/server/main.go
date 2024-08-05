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
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/emersion/go-smtp"
	"github.com/ksdme/mail/internal/backend"
	"github.com/ksdme/mail/internal/bus"
	"github.com/ksdme/mail/internal/config"
	"github.com/ksdme/mail/internal/models"
	"github.com/ksdme/mail/internal/tui"
	"github.com/ksdme/mail/internal/tui/colors"
	"github.com/ksdme/mail/internal/utils"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

func main() {
	if config.Settings.Debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	sqldb, err := sql.Open("sqlite3", config.Settings.DBURI)
	if err != nil {
		log.Panicf("opening db failed: %v", err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New())

	// Create the database tables if needed.
	// TODO: Have an actual migration system.
	if config.Settings.DBMigrate {
		slog.Info("creating tables")
		ctx := context.Background()
		utils.MustExec(db.NewCreateTable().Model(&models.Account{}).Exec(ctx))
		utils.MustExec(db.NewCreateTable().Model(&models.Mailbox{}).Exec(ctx))
		utils.MustExec(db.NewCreateTable().Model(&models.Mail{}).Exec(ctx))
	}

	// Start the servers and workers.
	var wg sync.WaitGroup

	wg.Add(3)
	go runSSHServer(db, &wg)
	go runMailServer(db, &wg)
	go runCleanupWorker(db, &wg)

	wg.Wait()
}

func runMailServer(db *bun.DB, wg *sync.WaitGroup) {
	defer wg.Done()

	server := smtp.NewServer(backend.NewBackend(db))
	server.Addr = config.Settings.SMTPBindAddr
	server.Domain = config.Settings.MXHost

	slog.Info("starting smtp server", "at", config.Settings.SMTPBindAddr)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("failed serving smtp server", "err", err)
	}
}

func runSSHServer(db *bun.DB, wg *sync.WaitGroup) {
	defer wg.Done()

	options := []ssh.Option{
		wish.WithAddress(config.Settings.SSHBindAddr),
		wish.WithHostKeyPath(config.Settings.SSHHostKeyPath),
	}

	if config.Settings.SSHAuthorizedKeysPath == "" {
		options = append(options, wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			return true
		}))
	} else {
		if _, err := os.Stat(config.Settings.SSHAuthorizedKeysPath); err != nil {
			panic(errors.Wrap(err, "could not stat authorized_keys file"))
		}
		options = append(options, wish.WithAuthorizedKeys(config.Settings.SSHAuthorizedKeysPath))
	}

	options = append(options, wish.WithMiddleware(
		// Run the bubbletea program.
		bubbletea.Middleware(func(session ssh.Session) (tea.Model, []tea.ProgramOption) {
			renderer := bubbletea.MakeRenderer(session)
			slog.Info(
				"client configured with",
				"color-profile", renderer.ColorProfile(),
				"has-dark-background", renderer.HasDarkBackground(),
			)

			// TODO: We should adjust the color palette based on color profile.
			palette := colors.DefaultColorPalette()
			if !renderer.HasDarkBackground() {
				palette = colors.DefaultLightColorPalette()
			}

			account := session.Context().Value("account").(models.Account)
			model := tui.NewModel(db, account, palette)
			options := []tea.ProgramOption{tea.WithAltScreen()}
			return model, options
		}),

		// Resolve the account.
		func(next ssh.Handler) ssh.Handler {
			return func(session ssh.Session) {
				account, err := models.GetOrCreateAccountFromPublicKey(
					session.Context(),
					db,
					session.PublicKey(),
				)
				if err != nil {
					io.WriteString(session, err.Error()+"\n")
					return
				}

				session.Context().SetValue("account", *account)
				next(session)

				bus.MailboxContentsUpdatedSignal.CleanUp(account.ID)
				slog.Debug("cleaning up mailbox signals", "account", account.ID)
			}
		},

		// Log the request.
		func(next ssh.Handler) ssh.Handler {
			return func(s ssh.Session) {
				at := time.Now()
				slog.Info("client connected",
					"at", at,
					"user", s.User(),
					"client", s.Context().ClientVersion(),
				)
				next(s)
				slog.Info(
					"client disconnected",
					"user", s.User(),
					"alive", time.Since(at),
					"at", at,
				)
			}
		},

		// Only allow active terminals.
		activeterm.Middleware(),
	))

	server, err := wish.NewServer(options...)
	if err != nil {
		slog.Error("failed creating ssh server", "err", err)
	}

	slog.Info("starting ssh server", "at", config.Settings.SSHBindAddr)
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

func must(result sql.Result, err error) sql.Result {
	if err != nil {
		log.Panicf("could not run query: %v", err)
	}
	return result
}
