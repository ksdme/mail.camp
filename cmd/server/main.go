package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/ksdme/mail/internal/apps"
	accountmodels "github.com/ksdme/mail/internal/apps/accounts/models"
	"github.com/ksdme/mail/internal/apps/clipboard"
	clipboardmodels "github.com/ksdme/mail/internal/apps/clipboard/models"
	"github.com/ksdme/mail/internal/apps/mail"
	mailmodels "github.com/ksdme/mail/internal/apps/mail/models"
	"github.com/ksdme/mail/internal/config"
	"github.com/ksdme/mail/internal/core"
	"github.com/ksdme/mail/internal/core/tui/colors"
	"github.com/ksdme/mail/internal/core/tui/menu"
	"github.com/ksdme/mail/internal/utils"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

func main() {
	if config.Core.Debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	slog.Info("starting up")

	sqldb, err := sql.Open("sqlite3", config.Core.DBURI)
	if err != nil {
		log.Panicf("opening db failed: %v", err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New())

	// Create the database tables if needed.
	// TODO: Have an actual migration system.
	if config.Core.DBMigrate {
		slog.Info("creating tables")
		ctx := context.Background()
		utils.MustExec(db.NewCreateTable().Model(&accountmodels.Account{}).Exec(ctx))
		utils.MustExec(db.NewCreateTable().Model(&mailmodels.Mailbox{}).Exec(ctx))
		utils.MustExec(db.NewCreateTable().Model(&mailmodels.Mail{}).Exec(ctx))
		utils.MustExec(db.NewCreateTable().Model(&clipboardmodels.ClipboardItem{}).Exec(ctx))
	}

	apps := []core.App{}
	if config.Core.MailAppEnabled {
		apps = append(apps, &mail.App{
			DB: db,
		})
	}
	if config.Core.ClipboardAppEnabled {
		apps = append(apps, &clipboard.App{
			DB: db,
		})
	}
	if len(apps) == 0 {
		log.Panicf("no app is enabled")
	}

	for _, app := range apps {
		name, _, _ := app.Info()
		slog.Info("enabling app", "name", name)
	}
	for _, app := range apps {
		app.Init()
	}
	for _, app := range apps {
		defer app.CleanUp()
	}

	startSSHServer(db, apps)
}

func startSSHServer(db *bun.DB, enabledApps []core.App) {
	options := []ssh.Option{
		wish.WithAddress(config.Core.SSHBindAddr),
		wish.WithHostKeyPath(config.Core.SSHHostKeyPath),
	}

	// Set up access limitations.
	if config.Core.SSHAuthorizedKeysPath == "" {
		options = append(options, wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			return true
		}))
	} else {
		if _, err := os.Stat(config.Core.SSHAuthorizedKeysPath); err != nil {
			panic(errors.Wrap(err, "could not stat authorized_keys file"))
		}
		options = append(options, wish.WithAuthorizedKeys(config.Core.SSHAuthorizedKeysPath))
	}

	options = append(options, wish.WithMiddleware(
		// Determine which application is being requested and route the connection to it.
		// Prefers presenting a tui, but falls back to a non-tui interface if the connection
		// or the application doesn't support it.
		handleIncoming(enabledApps),

		// Resolve the account.
		func(next ssh.Handler) ssh.Handler {
			return func(s ssh.Session) {
				account, err := accountmodels.GetOrCreateAccountFromPublicKey(
					s.Context(),
					db,
					s.PublicKey(),
				)
				if err != nil {
					fmt.Fprintln(s, err.Error())
					s.Exit(1)
					return
				}

				s.Context().SetValue("account", *account)
				next(s)
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
	))

	server, err := wish.NewServer(options...)
	if err != nil {
		slog.Error("failed creating ssh server", "err", err)
	}

	slog.Info("starting ssh server", "at", config.Core.SSHBindAddr)
	if err = server.ListenAndServe(); err != nil {
		slog.Error("failed serving ssh connections", "err", err)
	}
}

func handleIncoming(enabledApps []core.App) wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			account := s.Context().Value("account").(accountmodels.Account)
			stderr := s.Stderr()

			pty, _, active := s.Pty()
			renderer := bubbletea.MakeRenderer(s)
			slog.Info(
				"client configured with",
				"active", active,
				"term", pty.Term,
				"has-dark-background", renderer.HasDarkBackground(),
			)

			// TODO: We should adjust the color palette based on color profile.
			palette := colors.DefaultColorDarkPalette()
			if !renderer.HasDarkBackground() {
				palette = colors.DefaultLightColorPalette()
			}

			// Show a menu if no app was explicitly requested.
			command := s.Command()
			if len(command) == 0 {
				utils.RunTeaInSession(
					next,
					s,
					menu.NewModel(
						enabledApps,
						s,
						account,
						renderer,
						palette,
					),
				)
				return
			}

			// The client invocation arguments.
			var args apps.AppArgs
			if retcode, consumed := utils.ParseArgs(s, "ssh.camp", command, &args); consumed {
				s.Exit(retcode)
				return
			}

			// TODO: Eh, figure out a way to not have to do this.
			var name string
			switch {
			case args.Mail != nil:
				name = "mail"

			case args.Clipboard != nil:
				name = "clipboard"
			}

			// Figure out which app needs to be run.
			var app core.App = nil
			for _, element := range enabledApps {
				current, _, _ := element.Info()
				if current == name {
					app = element
					break
				}
			}
			// Technically, because the argument parser handles validating the available
			// app names, we should not reach to this point unless the switch above is
			// incomplete.
			if app == nil {
				if len(name) == 0 {
					fmt.Fprintf(stderr, "unknown app '%s'\n", name)
				} else {
					fmt.Fprintf(stderr, "%s app is disabled\n", name)
				}
				s.Exit(1)
				return
			}

			if retcode, err := app.HandleRequest(next, s, args, account, active, renderer, palette); err != nil {
				slog.Error("could not process the request", "app", "n", "account", account.ID, "err", err, "args", command)

				err = errors.Wrap(err, "could not process your request")
				fmt.Fprintln(stderr, err.Error())

				if retcode <= 0 {
					retcode = 1
				}
				s.Exit(retcode)
			} else {
				s.Exit(retcode)
			}
		}
	}
}
