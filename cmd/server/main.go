package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/ksdme/mail/internal/apps"
	mail "github.com/ksdme/mail/internal/apps/mail/models"
	"github.com/ksdme/mail/internal/apps/mail/tui"
	"github.com/ksdme/mail/internal/config"
	core "github.com/ksdme/mail/internal/core/models"
	"github.com/ksdme/mail/internal/core/tui/colors"
	"github.com/ksdme/mail/internal/utils"
	_ "github.com/mattn/go-sqlite3"
	"github.com/muesli/termenv"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

func main() {
	if config.Core.Debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

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
		utils.MustExec(db.NewCreateTable().Model(&core.Account{}).Exec(ctx))
		utils.MustExec(db.NewCreateTable().Model(&mail.Mailbox{}).Exec(ctx))
		utils.MustExec(db.NewCreateTable().Model(&mail.Mail{}).Exec(ctx))
	}

	enabledApps := apps.EnabledApps(db)
	if len(enabledApps) == 0 {
		log.Panicf("no app is enabled")
	}
	for _, app := range enabledApps {
		name, _, _ := app.Info()
		slog.Info("enabling app", "name", name)
	}

	// Start serving.
	for _, app := range enabledApps {
		app.Init()
	}
	startSSHServer(db, enabledApps)
	for _, app := range enabledApps {
		app.CleanUp()
	}
}

func startSSHServer(db *bun.DB, enabledApps []apps.App) {
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
		handleIncoming(db, enabledApps),

		// Resolve the account.
		func(next ssh.Handler) ssh.Handler {
			return func(s ssh.Session) {
				account, err := core.GetOrCreateAccountFromPublicKey(
					s.Context(),
					db,
					s.PublicKey(),
				)
				if err != nil {
					utils.WriteStringToSSH(s, err.Error())
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

func handleIncoming(db *bun.DB, enabledApps []apps.App) wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			pty, _, active := s.Pty()
			renderer := bubbletea.MakeRenderer(s)
			slog.Info(
				"client configured with",
				"active", active,
				"term", pty.Term,
				"has-dark-background", renderer.HasDarkBackground(),
			)

			// Show a menu if no app was explicitly requested.
			commands := s.Command()
			if len(commands) == 0 {
				utils.WriteStringToSSH(s, "todo: implement menu tui")
				s.Exit(1)
				return
			}

			// The client invocation arguments.
			name := strings.ToLower(commands[0])
			args := commands[1:]

			// Figure out which app needs to be run.
			var app apps.App = nil
			for _, element := range enabledApps {
				current, _, _ := element.Info()
				if current == name {
					app = element
					break
				}
			}
			if app == nil {
				// TODO: Mention the available names.
				utils.WriteStringToSSH(s, "todo: could not find an app")
				s.Exit(1)
				return
			}

			// Try to run the tui mode of the application.
			account := s.Context().Value("account").(core.Account)
			if active {
				// TODO: We should adjust the color palette based on color profile.
				palette := colors.DefaultColorDarkPalette()
				if !renderer.HasDarkBackground() {
					palette = colors.DefaultLightColorPalette()
				}

				program, cleanup, err := app.HandleTUI(args, account, renderer, palette)
				if err != nil {
					utils.WriteStringToSSH(s, "todo: something unexpected happened while routing your request")
					s.Exit(1)
					return
				}
				if program != nil {
					slog.Debug("running tui program", "app", name)

					if cleanup != nil {
						defer cleanup()
					}

					// Run the tui application. Piggy back of the official middleware to handle that bit.
					middleware := bubbletea.MiddlewareWithColorProfile(func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
						model := tui.NewModel(db, account, renderer, palette)
						options := []tea.ProgramOption{tea.WithAltScreen()}
						return model, options
					}, termenv.ANSI)
					middleware(next)(s)

					return
				}
			}

			// Run the non-tui mode of the application.
			handler, cleanup, err := app.Handle(args, s, account)
			if err != nil {
				utils.WriteStringToSSH(s, "todo: something unexpected happened while routing your request")
				s.Exit(1)
				return
			}
			if handler == nil {
				utils.WriteStringToSSH(s, "todo: something unexpected happened while processing your request")
				s.Exit(1)
				return
			}
			if cleanup != nil {
				defer cleanup()
			}
			handler()
		}
	}
}
